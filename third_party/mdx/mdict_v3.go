package mdx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/adler32"
	"io"
	"math"
	"os"

	"github.com/cespare/xxhash/v2"
	"github.com/rasky/go-lzo"
	"golang.org/x/crypto/salsa20/salsa"
)

const (
	v3BlockRecordData  uint32 = 0x01000000
	v3BlockRecordIndex uint32 = 0x02000000
	v3BlockKeyData     uint32 = 0x03000000
	v3BlockKeyIndex    uint32 = 0x04000000
)

// deriveV3Key builds the engine 3.0 block key from a header UUID.
// MDXBuilder 4.0 RC4 and later store xxh64(uuid[:mid]) || xxh64(uuid[mid:]).
func deriveV3Key(uuid []byte) []byte {
	mid := (len(uuid) + 1) / 2
	key := make([]byte, 16)
	binary.LittleEndian.PutUint64(key[:8], xxhash.Sum64(uuid[:mid]))
	binary.LittleEndian.PutUint64(key[8:], xxhash.Sum64(uuid[mid:]))
	return key
}

// loadFormatV3 reads engine 3.0 typed blocks into the key list and record map.
func (mdict *MdictBase) loadFormatV3() error {
	file, err := os.Open(mdict.filePath)
	if err != nil {
		return fmt.Errorf("open mdict 3.0 file '%s': %w", mdict.filePath, err)
	}
	defer closeFile(file)

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat mdict 3.0 file '%s': %w", mdict.filePath, err)
	}
	if _, err := file.Seek(mdict.meta.keyBlockMetaStartOffset, io.SeekStart); err != nil {
		return fmt.Errorf("seek mdict 3.0 blocks in '%s': %w", mdict.filePath, err)
	}

	var keyDataOffset, recordDataOffset int64 = -1, -1
	for {
		var blockType uint32
		err := binary.Read(file, binary.BigEndian, &blockType)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read mdict 3.0 block type in '%s': %w", mdict.filePath, err)
		}
		var blockSize uint64
		if err := binary.Read(file, binary.BigEndian, &blockSize); err != nil {
			return fmt.Errorf("read mdict 3.0 block size in '%s': %w", mdict.filePath, err)
		}
		if blockSize > uint64(math.MaxInt64) {
			return fmt.Errorf("mdict 3.0 block in '%s' is too large", mdict.filePath)
		}
		payloadOffset, err := file.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("tell mdict 3.0 block in '%s': %w", mdict.filePath, err)
		}
		if payloadOffset+int64(blockSize) > info.Size() {
			return fmt.Errorf("mdict 3.0 block in '%s' extends past end of file", mdict.filePath)
		}
		switch blockType {
		case v3BlockRecordData:
			recordDataOffset = payloadOffset
		case v3BlockKeyData:
			keyDataOffset = payloadOffset
		case v3BlockRecordIndex, v3BlockKeyIndex:
		default:
			return fmt.Errorf("unknown mdict 3.0 block type %#x in '%s'", blockType, mdict.filePath)
		}
		if _, err := file.Seek(int64(blockSize), io.SeekCurrent); err != nil {
			return fmt.Errorf("skip mdict 3.0 block in '%s': %w", mdict.filePath, err)
		}
	}
	if keyDataOffset < 0 || recordDataOffset < 0 {
		return fmt.Errorf("mdict 3.0 file '%s' is missing key or record data", mdict.filePath)
	}

	entries, err := readV3KeyEntries(file, keyDataOffset, info.Size(), mdict)
	if err != nil {
		return err
	}
	records, err := readV3RecordBlocks(file, recordDataOffset, info.Size())
	if err != nil {
		return err
	}

	mdict.keyBlockData = &mdictKeyBlockData{
		keyEntries:     entries,
		keyEntriesSize: int64(len(entries)),
	}
	mdict.recordBlockInfo = &mdictRecordBlockInfo{
		recordInfoList: records,
	}
	mdict.recordBlockMeta = &mdictRecordBlockMeta{
		entriesNum:     int64(len(entries)),
		recordBlockNum: int64(len(records)),
	}
	return nil
}

func readV3SectionHeader(file *os.File, offset, fileSize int64) (count uint32, payload int64, err error) {
	if offset < 0 || offset+12 > fileSize {
		return 0, 0, fmt.Errorf("mdict 3.0 section header at %d is outside the file", offset)
	}
	var head [12]byte
	if _, err := file.ReadAt(head[:], offset); err != nil {
		return 0, 0, fmt.Errorf("read mdict 3.0 section header at %d: %w", offset, err)
	}
	count = binary.BigEndian.Uint32(head[:4])
	return count, offset + 12, nil
}

func readV3KeyEntries(file *os.File, offset, fileSize int64, mdict *MdictBase) ([]*MDictKeywordEntry, error) {
	count, pos, err := readV3SectionHeader(file, offset, fileSize)
	if err != nil {
		return nil, err
	}
	var entries []*MDictKeywordEntry
	for i := uint32(0); i < count; i++ {
		decompSize, _, blob, next, err := readV3SizedBlock(file, pos, fileSize)
		if err != nil {
			return nil, fmt.Errorf("mdict 3.0 key block %d: %w", i, err)
		}
		plain, err := decodeV3Block(blob, int64(decompSize), mdict.meta.v3Key)
		if err != nil {
			return nil, fmt.Errorf("mdict 3.0 key block %d: %w", i, err)
		}
		entries = append(entries, mdict.splitKeyBlock(plain)...)
		pos = next
	}
	return entries, nil
}

func readV3RecordBlocks(file *os.File, offset, fileSize int64) ([]*MdictRecordBlockInfoListItem, error) {
	count, pos, err := readV3SectionHeader(file, offset, fileSize)
	if err != nil {
		return nil, err
	}
	list := make([]*MdictRecordBlockInfoListItem, 0, count)
	var logical int64
	for i := uint32(0); i < count; i++ {
		decompSize, compSize, _, next, err := readV3SizedBlock(file, pos, fileSize)
		if err != nil {
			return nil, fmt.Errorf("mdict 3.0 record block %d: %w", i, err)
		}
		blobOffset := pos + 8
		list = append(list, &MdictRecordBlockInfoListItem{
			compressSize:                int64(compSize),
			deCompressSize:              int64(decompSize),
			compressAccumulatorOffset:   blobOffset,
			deCompressAccumulatorOffset: logical,
		})
		logical += int64(decompSize)
		pos = next
	}
	return list, nil
}

func readV3SizedBlock(file *os.File, pos, fileSize int64) (decompSize, compSize uint32, blob []byte, next int64, err error) {
	if pos < 0 || pos+8 > fileSize {
		return 0, 0, nil, 0, fmt.Errorf("block header at %d is outside the file", pos)
	}
	var head [8]byte
	if _, err = file.ReadAt(head[:], pos); err != nil {
		return 0, 0, nil, 0, err
	}
	decompSize = binary.BigEndian.Uint32(head[:4])
	compSize = binary.BigEndian.Uint32(head[4:])
	blobOffset := pos + 8
	if blobOffset+int64(compSize) > fileSize {
		return 0, 0, nil, 0, fmt.Errorf("block at %d extends past end of file", pos)
	}
	blob = make([]byte, compSize)
	if compSize > 0 {
		if _, err = file.ReadAt(blob, blobOffset); err != nil {
			return 0, 0, nil, 0, err
		}
	}
	return decompSize, compSize, blob, blobOffset + int64(compSize), nil
}

func decodeV3Block(block []byte, decompressedSize int64, fileKey []byte) ([]byte, error) {
	if len(block) < 8 {
		return nil, fmt.Errorf("block is too short (%d bytes)", len(block))
	}
	info := binary.LittleEndian.Uint32(block[:4])
	compression := info & 0xf
	encryption := (info >> 4) & 0xf
	encryptionSize := int((info >> 8) & 0xff)
	expected := binary.BigEndian.Uint32(block[4:8])

	payload := append([]byte(nil), block[8:]...)
	if encryptionSize > len(payload) {
		return nil, fmt.Errorf("encryption size %d exceeds block data %d", encryptionSize, len(payload))
	}
	if encryption != 0 {
		key := fileKey
		if len(key) == 0 {
			key = ripemd128bytes(block[4:8])
		}
		switch encryption {
		case 1:
			fastDecrypt(payload[:encryptionSize], key, int64(encryptionSize), int64(len(key)))
		case 2:
			if err := salsa208XOR(payload[:encryptionSize], key); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("encryption method %d is not supported", encryption)
		}
	}
	if adler32.Checksum(payload) != expected {
		return nil, fmt.Errorf("block checksum mismatch")
	}

	var plain []byte
	switch compression {
	case 0:
		plain = payload
	case 1:
		out, err := lzo.Decompress1X(bytes.NewReader(payload), 0, int(decompressedSize))
		if err != nil {
			return nil, fmt.Errorf("LZO decompression failed: %w", err)
		}
		plain = out
	case 2:
		out, err := zlibDecompress(payload, 0, int64(len(payload)))
		if err != nil {
			return nil, fmt.Errorf("zlib decompression failed: %w", err)
		}
		plain = out
	default:
		return nil, fmt.Errorf("compression method %d is not supported", compression)
	}
	if int64(len(plain)) != decompressedSize {
		return nil, fmt.Errorf("decompressed size mismatch: expected %d, got %d", decompressedSize, len(plain))
	}
	return plain, nil
}

func (mdict *MdictBase) locateV3Record(index *MDictKeywordIndex) ([]byte, error) {
	block, err := mdict.readAndDecodeV3Block(
		index.RecordBlock.DataStartOffset,
		index.RecordBlock.CompressSize,
		index.RecordBlock.DeCompressSize,
	)
	if err != nil {
		return nil, fmt.Errorf("decode mdict 3.0 record for '%s': %w", index.KeywordEntry.KeyWord, err)
	}
	start := index.RecordBlock.KeyWordPartStartOffset
	end := index.RecordBlock.KeyWordPartDataEndOffset
	if start < 0 || end < start || end > int64(len(block)) {
		return nil, fmt.Errorf("record slice for '%s' is out of range: start=%d end=%d len=%d",
			index.KeywordEntry.KeyWord, start, end, len(block))
	}
	return block[start:end], nil
}

func (mdict *MdictBase) readAndDecodeV3Block(offset, compressedSize, decompressedSize int64) ([]byte, error) {
	if compressedSize < 8 {
		return nil, fmt.Errorf("compressed block at %d is too small (%d)", offset, compressedSize)
	}
	file, err := os.Open(mdict.filePath)
	if err != nil {
		return nil, err
	}
	defer closeFile(file)
	buf := make([]byte, compressedSize)
	if _, err := file.ReadAt(buf, offset); err != nil {
		return nil, err
	}
	return decodeV3Block(buf, decompressedSize, mdict.meta.v3Key)
}

// salsa208XOR applies Salsa20/8 with an 8-byte zero nonce, matching MDXBuilder 4.x.
func salsa208XOR(data, key []byte) error {
	if len(key) != 16 && len(key) != 32 {
		return fmt.Errorf("salsa20 key must be 16 or 32 bytes, got %d", len(key))
	}
	var counter uint64
	for offset := 0; offset < len(data); {
		var in, out [64]byte
		writeSalsaState(&in, key, counter)
		salsa.Core208(&out, &in)
		n := 64
		if remain := len(data) - offset; remain < n {
			n = remain
		}
		for i := 0; i < n; i++ {
			data[offset+i] ^= out[i]
		}
		counter++
		offset += n
	}
	return nil
}

func writeSalsaState(dst *[64]byte, key []byte, counter uint64) {
	constants := []byte("expand 32-byte k")
	key2 := key[16:]
	key1 := key[:16]
	if len(key) == 16 {
		constants = []byte("expand 16-byte k")
		key2 = key
	}
	copy(dst[0:4], constants[0:4])
	copy(dst[4:20], key1)
	copy(dst[20:24], constants[4:8])
	binary.LittleEndian.PutUint64(dst[32:40], counter)
	copy(dst[40:44], constants[8:12])
	copy(dst[44:60], key2)
	copy(dst[60:64], constants[12:16])
}
