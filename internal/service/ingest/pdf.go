package ingest

import (
	"fmt"
	"io"
	"strings"

	pdf "github.com/ledongthuc/pdf"
)

func ExtractPDFText(path string) (string, error) {
	file, reader, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("open pdf %q: %w", path, err)
	}
	defer file.Close()

	plainReader, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("extract plain text from %q: %w", path, err)
	}

	content, err := io.ReadAll(plainReader)
	if err != nil {
		return "", fmt.Errorf("read plain text from %q: %w", path, err)
	}

	return strings.TrimSpace(string(content)), nil
}
