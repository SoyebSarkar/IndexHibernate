package snapshot

import (
	"io"
	"os"
	"path/filepath"
)

func SaveSchema(baseDir string, schema []byte) error {
	path := filepath.Join(baseDir, "schema.json")
	os.MkdirAll(filepath.Dir(path), 0755)

	return os.WriteFile(path, schema, 0644)
}

func SaveDocuments(baseDir string, r io.Reader) error {
	path := filepath.Join(baseDir, "documents.jsonl")

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}
