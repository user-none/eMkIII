package romloader

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/nwaples/rardecode/v2"
)

// extractFromRAR extracts the first .sms file from a RAR archive
func extractFromRAR(path string) ([]byte, string, error) {
	r, err := rardecode.OpenReader(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open rar: %w", err)
	}
	defer r.Close()

	for {
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("failed to read rar entry: %w", err)
		}

		if header.IsDir {
			continue
		}
		if !isSMSFile(header.Name) {
			continue
		}

		data, err := limitedRead(r)
		if err != nil {
			return nil, "", fmt.Errorf("failed to read %s: %w", header.Name, err)
		}
		return data, filepath.Base(header.Name), nil
	}

	return nil, "", ErrNoSMSFile
}
