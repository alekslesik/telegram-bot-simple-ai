package ingest

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

type Service struct{}

type InputFile struct {
	Path    string
	Name    string
	ModTime time.Time
}

func New() *Service {
	return &Service{}
}

func (s *Service) PrepareInputs(root string) ([]InputFile, error) {
	files := make([]InputFile, 0)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".pdf") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}

		files = append(files, InputFile{
			Path:    path,
			Name:    d.Name(),
			ModTime: info.ModTime(),
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan root %s: %w", root, err)
	}

	return SortInputFiles(files), nil
}
