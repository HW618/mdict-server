package dict

import (
	"testing"
)

func TestExtractStyleBlocks_MultiLineCSS(t *testing.T) {
	html := `<html><head>
<style type="text/css">
  .definition {
    font-family: Arial;
    font-size: 14px;
    color: #333;
  }
  .example {
    margin: 10px;
    padding: 5px;
  }
</style>
</head><body><div class="definition">hello</div></body></html>`

	blocks := extractStyleBlocks(html)
	if len(blocks) == 0 {
		t.Fatal("expected to extract style blocks from multi-line CSS, got 0")
	}

	css := blocks[0]
	if !containsStr(css, "font-family") {
		t.Errorf("expected CSS to contain 'font-family', got: %s", css)
	}
	if !containsStr(css, "font-size") {
		t.Errorf("expected CSS to contain 'font-size', got: %s", css)
	}
	if !containsStr(css, ".example") {
		t.Errorf("expected CSS to contain '.example', got: %s", css)
	}
}

func TestExtractStyleBlocks_LinkStylesheet(t *testing.T) {
	html := `<html><head>
<link rel="stylesheet" href="oald10.css" type="text/css">
<link rel="stylesheet" href="mwa.css">
</head><body><div>hello</div></body></html>`

	blocks := extractStyleBlocks(html)
	if len(blocks) < 2 {
		t.Fatalf("expected at least 2 blocks from link tags, got %d", len(blocks))
	}

	if !containsStr(blocks[0], "@import") {
		t.Errorf("expected first block to be @import, got: %s", blocks[0])
	}
	if !containsStr(blocks[0], "oald10.css") {
		t.Errorf("expected first block to reference oald10.css, got: %s", blocks[0])
	}
	if !containsStr(blocks[1], "mwa.css") {
		t.Errorf("expected second block to reference mwa.css, got: %s", blocks[1])
	}
}

func TestRewriteAssetURLs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		dictID   string
		expected string
	}{
		{
			name:     "href CSS",
			input:    `<link rel="stylesheet" href="style.css">`,
			dictID:   "abc123",
			expected: `/api/v1/assets/abc123/style.css`,
		},
		{
			name:     "src image",
			input:    `<img src="images/photo.png">`,
			dictID:   "abc123",
			expected: `/api/v1/assets/abc123/images/photo.png`,
		},
		{
			name:     "CSS url() font",
			input:    `url("fonts/MyFont.woff2")`,
			dictID:   "abc123",
			expected: `/api/v1/assets/abc123/fonts/MyFont.woff2`,
		},
		{
			name:     "CSS @import",
			input:    `@import "oald10.css";`,
			dictID:   "abc123",
			expected: `/api/v1/assets/abc123/oald10.css`,
		},
		{
			name:     "absolute URL unchanged",
			input:    `<img src="https://example.com/img.png">`,
			dictID:   "abc123",
			expected: `https://example.com/img.png`,
		},
		{
			name:     "already rewritten URL unchanged",
			input:    `<img src="/api/v1/assets/abc123/img.png">`,
			dictID:   "abc123",
			expected: `/api/v1/assets/abc123/img.png`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rewriteAssetURLs(tt.input, tt.dictID)
			if !containsStr(result, tt.expected) {
				t.Errorf("expected result to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
