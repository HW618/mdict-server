package mdx

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/adler32"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestSalsa208MatchesIndependentVector(t *testing.T) {
	key := bytes.Repeat([]byte{0x11}, 16)
	plain := []byte("hello world!! this crosses sixty-four bytes boundary for salsa20/8!!")
	got := append([]byte(nil), plain...)
	if err := salsa208XOR(got, key); err != nil {
		t.Fatal(err)
	}
	const want = "bcf164174f17525be11d504a235fdbd2311cfa3d824cb61cb8d522721356b5c0d1f48a8adb90481caf785e20379d57ff0003d91d5cd5a90c712be6aed97c872c3284adbe"
	if hexBytes(got) != want {
		t.Fatalf("salsa20/8 keystream mismatch:\n got %s\nwant %s", hexBytes(got), want)
	}
}

func TestFormatV3PlainMDX(t *testing.T) {
	path := writeV3Dict(t, "plain.mdx", "", v3CompNone, v3EncNone, []v3Entry{
		{key: "apple", value: "<p>red fruit</p>"},
		{key: "汉语", value: "<p>Chinese</p>"},
	}, false)
	assertLookup(t, path, "apple", "<p>red fruit</p>")
	assertLookup(t, path, "汉语", "<p>Chinese</p>")
}

func TestFormatV3FastXORWithoutUUID(t *testing.T) {
	path := writeV3Dict(t, "fast.mdx", "", v3CompZlib, v3EncFast, []v3Entry{
		{key: "apple", value: "<p>red fruit</p>"},
		{key: "zebra", value: "<p>striped</p>"},
	}, true)
	assertLookup(t, path, "apple", "<p>red fruit</p>")
	assertLookup(t, path, "zebra", "<p>striped</p>")

	corrupted := filepath.Join(t.TempDir(), "broken.mdx")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)-1] ^= 0xff
	if err := os.WriteFile(corrupted, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	dict, err := New(corrupted)
	if err != nil {
		t.Fatal(err)
	}
	if err := dict.BuildIndex(); err != nil {
		t.Fatal(err)
	}
	if _, err := dict.Lookup("zebra"); err == nil {
		t.Fatal("expected checksum failure after corrupting the last record block")
	}
}

func TestFormatV3SalsaWithUUID(t *testing.T) {
	const uuid = "0123456789abcdef0123456789abcdef"
	value := strings.Repeat("definition-", 30)
	path := writeV3Dict(t, "salsa.mdx", uuid, v3CompZlib, v3EncSalsa, []v3Entry{
		{key: "apple", value: "<p>red fruit</p>"},
		{key: "long", value: value},
	}, true)
	assertLookup(t, path, "apple", "<p>red fruit</p>")
	assertLookup(t, path, "long", value)
	dict, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := dict.BuildIndex(); err != nil {
		t.Fatal(err)
	}
	if got := dict.DictionaryInfo().EntryCount; got != 2 {
		t.Fatalf("entry count = %d, want 2", got)
	}
	if !bytes.Equal(dict.meta.v3Key, deriveV3Key([]byte(uuid))) {
		t.Fatal("UUID did not produce the engine 3.0 block key")
	}
}

func TestFormatV3MDDUsesUTF8Keys(t *testing.T) {
	path := writeV3Dict(t, "assets.mdd", "", v3CompZlib, v3EncNone, []v3Entry{
		{key: `\style.css`, value: "body{}"},
	}, false)
	assertLookup(t, path, `\style.css`, "body{}")
}

func TestFormatV2FixtureStillLoads(t *testing.T) {
	path := filepath.Join("..", "..", "internal", "dict", "testdata", "primary.mdx")
	assertLookup(t, path, "apple", "<p>apple</p>\x00")
	assertLookup(t, path, "中文", "<p>中文</p>\x00")
}

func fastEncrypt(data, key []byte) {
	previous := byte(0x36)
	for i := 0; i < len(data); i++ {
		mixed := data[i] ^ previous ^ byte(i&0xff) ^ key[i%len(key)]
		cipher := byte((mixed >> 4) | (mixed << 4))
		previous = cipher
		data[i] = cipher
	}
}

const (
	v3CompNone = 0
	v3CompZlib = 2
	v3EncNone  = 0
	v3EncFast  = 1
	v3EncSalsa = 2
)

type v3Entry struct {
	key   string
	value string
}

func writeV3Dict(t *testing.T, name, uuid string, compression, encryption int, entries []v3Entry, splitRecords bool) string {
	t.Helper()
	var fileKey []byte
	if uuid != "" {
		fileKey = deriveV3Key([]byte(uuid))
	}
	var keyPlain bytes.Buffer
	var recordParts [][]byte
	var logical uint64
	for _, entry := range entries {
		var off [8]byte
		binary.BigEndian.PutUint64(off[:], logical)
		keyPlain.Write(off[:])
		keyPlain.WriteString(entry.key)
		keyPlain.WriteByte(0)
		part := []byte(entry.value)
		recordParts = append(recordParts, part)
		logical += uint64(len(part))
	}

	keyBlock := encodeV3Block(t, keyPlain.Bytes(), compression, encryption, fileKey)
	var recordBlocks []encodedBlock
	if splitRecords && len(recordParts) > 1 {
		recordBlocks = append(recordBlocks, encodeV3Block(t, recordParts[0], compression, encryption, fileKey))
		recordBlocks = append(recordBlocks, encodeV3Block(t, bytes.Join(recordParts[1:], nil), compression, encryption, fileKey))
	} else {
		recordBlocks = append(recordBlocks, encodeV3Block(t, bytes.Join(recordParts, nil), compression, encryption, fileKey))
	}

	uuidAttr := ""
	if uuid != "" {
		uuidAttr = ` UUID="` + uuid + `"`
	}
	headerText := `<Dictionary GeneratedByEngineVersion="3.0" RequiredEngineVersion="2.0" Encrypted="No" Encoding="UTF-8" Format="Html" Title="v3"` + uuidAttr + "></Dictionary>\r\n"
	header := append(encodeUTF16LE(headerText), 0, 0)

	var body bytes.Buffer
	writeV3TypedBlock(&body, v3BlockKeyIndex, nil)
	writeV3TypedBlock(&body, v3BlockKeyData, v3Section(keyBlock))
	writeV3TypedBlock(&body, v3BlockRecordIndex, nil)
	writeV3TypedBlock(&body, v3BlockRecordData, v3Section(recordBlocks...))

	var file bytes.Buffer
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(header)))
	file.Write(length[:])
	file.Write(header)
	sum := adler32.Checksum(header)
	var checksum [4]byte
	binary.LittleEndian.PutUint32(checksum[:], sum)
	file.Write(checksum[:])
	file.Write(body.Bytes())

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, file.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

type encodedBlock struct {
	decompressed uint32
	bytes        []byte
}

func v3Section(blocks ...encodedBlock) []byte {
	var payload bytes.Buffer
	_ = binary.Write(&payload, binary.BigEndian, uint32(len(blocks)))
	var total uint64
	for _, block := range blocks {
		total += uint64(len(block.bytes))
	}
	_ = binary.Write(&payload, binary.BigEndian, total)
	for _, block := range blocks {
		_ = binary.Write(&payload, binary.BigEndian, block.decompressed)
		_ = binary.Write(&payload, binary.BigEndian, uint32(len(block.bytes)))
		payload.Write(block.bytes)
	}
	return payload.Bytes()
}

func encodeV3Block(t *testing.T, plain []byte, compression, encryption int, fileKey []byte) encodedBlock {
	t.Helper()
	body := append([]byte(nil), plain...)
	if compression == v3CompZlib {
		var buf bytes.Buffer
		writer := zlib.NewWriter(&buf)
		if _, err := writer.Write(plain); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		body = buf.Bytes()
	}
	sum := adler32.Checksum(body)
	var sumBytes [4]byte
	binary.BigEndian.PutUint32(sumBytes[:], sum)
	key := fileKey
	if len(key) == 0 && encryption != v3EncNone {
		key = ripemd128bytes(sumBytes[:])
	}
	stored := append([]byte(nil), body...)
	encSize := 0
	if encryption != v3EncNone {
		encSize = len(stored)
		if encSize > 255 {
			encSize = 255
		}
		switch encryption {
		case v3EncFast:
			fastEncrypt(stored[:encSize], key)
		case v3EncSalsa:
			if err := salsa208XOR(stored[:encSize], key); err != nil {
				t.Fatal(err)
			}
		default:
			t.Fatalf("unknown encryption %d", encryption)
		}
	}
	info := uint32(compression) | uint32(encryption)<<4 | uint32(encSize)<<8
	out := make([]byte, 8+len(stored))
	binary.LittleEndian.PutUint32(out[:4], info)
	binary.BigEndian.PutUint32(out[4:8], sum)
	copy(out[8:], stored)
	return encodedBlock{decompressed: uint32(len(plain)), bytes: out}
}

func writeV3TypedBlock(w *bytes.Buffer, blockType uint32, payload []byte) {
	_ = binary.Write(w, binary.BigEndian, blockType)
	_ = binary.Write(w, binary.BigEndian, uint64(len(payload)))
	w.Write(payload)
}

func encodeUTF16LE(text string) []byte {
	units := utf16.Encode([]rune(text))
	out := make([]byte, len(units)*2)
	for i, unit := range units {
		binary.LittleEndian.PutUint16(out[i*2:], unit)
	}
	return out
}

func assertLookup(t *testing.T, path, word, want string) {
	t.Helper()
	dict, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := dict.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex(%s): %v", path, err)
	}
	got, err := dict.Lookup(word)
	if err != nil {
		t.Fatalf("Lookup(%q): %v", word, err)
	}
	if string(got) != want {
		t.Fatalf("Lookup(%q) = %q, want %q", word, got, want)
	}
}

func hexBytes(data []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, len(data)*2)
	for i, b := range data {
		out[i*2] = digits[b>>4]
		out[i*2+1] = digits[b&0x0f]
	}
	return string(out)
}
