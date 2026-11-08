package mdx

import "testing"

func TestSplitKeyBlockUTF16TruncatedDoesNotPanic(t *testing.T) {
	// Odd-length tail used to panic on keyBlock[i+1] when scanning for UTF-16 NUL.
	block := []byte{
		0, 0, 0, 0, 0, 0, 0, 1, // record offset (8 bytes)
		'a', 0, 'b', // truncated UTF-16 key, no terminator
	}
	dict := &MdictBase{
		meta:     &mdictMeta{numberWidth: 8, encoding: EncodingUtf16, version: 2},
		fileType: MdictTypeMdx,
	}
	entries := dict.splitKeyBlock(block)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].RecordStartOffset != 1 {
		t.Fatalf("offset = %d, want 1", entries[0].RecordStartOffset)
	}
}

func TestSplitKeyBlockShortHeaderDoesNotPanic(t *testing.T) {
	dict := &MdictBase{
		meta:     &mdictMeta{numberWidth: 8, encoding: EncodingUtf8, version: 2},
		fileType: MdictTypeMdx,
	}
	entries := dict.splitKeyBlock([]byte{1, 2, 3})
	if len(entries) != 0 {
		t.Fatalf("entries = %d, want 0", len(entries))
	}
}
