package mdx

import "testing"

func TestLegacyUncompressedKeyBlockInfo(t *testing.T) {
	data := []byte{0, 0, 0, 1, 1, 0, 'a', 1, 0, 'z', 0, 0, 0, 10, 0, 0, 0, 20}
	d := &MdictBase{meta: &mdictMeta{version: 1.2, numberWidth: 4, encoding: EncodingUtf16}, keyBlockMeta: &mdictKeyBlockMeta{keyBlockNum: 1, entriesNum: 1, keyBlockDataTotalSize: 10, keyBlockInfoCompressedSize: int64(len(data))}}
	if err := d.decodeKeyBlockInfo(data); err != nil {
		t.Fatal(err)
	}
	if len(d.keyBlockInfo.keyBlockInfoList) != 1 || d.keyBlockInfo.keyBlockInfoList[0].keyBlockCompressSize != 10 {
		t.Fatalf("wrong block info: %+v", d.keyBlockInfo)
	}
}
