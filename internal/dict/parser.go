package dict

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	mdx "github.com/lib-x/mdx"
	"github.com/rs/zerolog/log"
)

// InitMDX initializes an MDX dictionary: open file, build index, return ready-to-use Mdict.
func InitMDX(mdxPath string) (*mdx.Mdict, error) {
	if _, err := os.Stat(mdxPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("dictionary file not found: %s", mdxPath)
	}

	dict, err := mdx.New(mdxPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open mdx: %w", err)
	}

	if err := dict.BuildIndex(); err != nil {
		return nil, fmt.Errorf("failed to build index: %w", err)
	}

	info := dict.DictionaryInfo()
	log.Info().
		Str("path", mdxPath).
		Str("title", info.Title).
		Int64("entries", info.EntryCount).
		Msg("MDX dictionary indexed")

	return dict, nil
}

// InitMDD initializes an MDD resource file: open file, build index, return ready-to-use Mdict.
func InitMDD(mddPath string) (*mdx.Mdict, error) {
	if _, err := os.Stat(mddPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("mdd file not found: %s", mddPath)
	}

	dict, err := mdx.New(mddPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open mdd: %w", err)
	}

	if err := dict.BuildIndex(); err != nil {
		return nil, fmt.Errorf("failed to build mdd index: %w", err)
	}

	info := dict.DictionaryInfo()
	log.Info().
		Str("path", mddPath).
		Int64("resources", info.EntryCount).
		Msg("MDD resource file indexed")

	return dict, nil
}

// GetMimeType returns the MIME type based on file extension.
func GetMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".m4a":
		return "audio/mp4"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".html", ".htm":
		return "text/html"
	case ".xml":
		return "application/xml"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
