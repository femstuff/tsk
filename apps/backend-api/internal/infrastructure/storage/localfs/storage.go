package localfs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	app "tsk/backend-api/internal/application/documentjob"
)

type Storage struct {
	root string
}

func New(root string) *Storage {
	return &Storage{root: filepath.Clean(root)}
}

func (s *Storage) Save(_ context.Context, namespace string, fileName string, content []byte) (app.StoredFile, error) {
	if len(content) == 0 {
		return app.StoredFile{}, errors.New("cannot store empty file")
	}

	namespace = sanitizeSegment(namespace, "misc")
	fileName = sanitizeSegment(fileName, "document.txt")
	relativePath := filepath.ToSlash(filepath.Join(namespace, time.Now().UTC().Format("20060102"), fileName))
	absolutePath := filepath.Join(s.root, filepath.FromSlash(relativePath))

	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		return app.StoredFile{}, err
	}

	if err := os.WriteFile(absolutePath, content, 0o644); err != nil {
		return app.StoredFile{}, err
	}

	return app.StoredFile{
		StorageKey: relativePath,
		FileName:   fileName,
		SizeBytes:  int64(len(content)),
	}, nil
}

func (s *Storage) Resolve(storageKey string) (string, error) {
	cleanKey := filepath.Clean(strings.TrimSpace(storageKey))
	if cleanKey == "." || cleanKey == "" {
		return "", errors.New("storage key is required")
	}

	absolutePath := filepath.Join(s.root, cleanKey)
	relativeToRoot, err := filepath.Rel(s.root, absolutePath)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(relativeToRoot, "..") {
		return "", errors.New("invalid storage key")
	}

	return absolutePath, nil
}

func sanitizeSegment(value string, fallback string) string {
	value = filepath.Base(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" || value == "." || value == "/" || value == `\` {
		return fallback
	}

	return value
}
