package filemanager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cs2admin/internal/pkg/logger"
)

const (
	maxFileSize = 10 * 1024 * 1024 // 10MB
)

var (
	ErrPathTraversal = errors.New("path traversal detected")
	ErrFileTooLarge  = errors.New("file exceeds 10MB limit")
)

// FileEntry represents a file or directory in the file browser.
type FileEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsDir    bool   `json:"is_dir"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

// validatePath resolves rootPath+relativePath, ensures the result is under rootPath, and returns the absolute path.
func validatePath(rootPath, relativePath string) (string, error) {
	rootAbs, err := filepath.Abs(rootPath)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}

	cleanRel := filepath.Clean(relativePath)
	if cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
		return "", ErrPathTraversal
	}

	joined := filepath.Join(rootAbs, cleanRel)
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	// Ensure result is under root (handles "a" vs "ab" by using Rel)
	rel, err := filepath.Rel(rootAbs, absJoined)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return "", ErrPathTraversal
	}

	resolved, err := filepath.EvalSymlinks(absJoined)
	if err != nil {
		if os.IsNotExist(err) {
			return absJoined, nil
		}
		return "", fmt.Errorf("resolve path: %w", err)
	}

	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	rel2, err := filepath.Rel(rootAbs, resolved)
	if err != nil || strings.HasPrefix(rel2, "..") || rel2 == ".." {
		return "", ErrPathTraversal
	}
	return resolved, nil
}

// ListDirectory lists files in the directory. relativePath is the path under rootPath.
func ListDirectory(rootPath, relativePath string) ([]FileEntry, error) {
	absPath, err := validatePath(rootPath, relativePath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory")
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	result := make([]FileEntry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			logger.Log.Debug().Err(err).Str("name", e.Name()).Msg("filemanager: skip entry")
			continue
		}

		rel, _ := filepath.Rel(rootPath, filepath.Join(absPath, e.Name()))
		rel = filepath.ToSlash(rel)

		entry := FileEntry{
			Name:     e.Name(),
			Path:     rel,
			IsDir:    e.IsDir(),
			Size:     0,
			Modified: info.ModTime().Format(time.RFC3339),
		}
		if !e.IsDir() {
			entry.Size = info.Size()
		}
		result = append(result, entry)
	}
	return result, nil
}

// ReadFile reads file content as string. Max 10MB.
func ReadFile(rootPath, relativePath string) (string, error) {
	absPath, err := validatePath(rootPath, relativePath)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("stat: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("is a directory")
	}
	if info.Size() > maxFileSize {
		return "", ErrFileTooLarge
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	return string(data), nil
}

// WriteFile writes content to file.
func WriteFile(rootPath, relativePath, content string) error {
	absPath, err := validatePath(rootPath, relativePath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

// CreateDirectory creates a directory.
func CreateDirectory(rootPath, relativePath string) error {
	absPath, err := validatePath(rootPath, relativePath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(absPath, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return nil
}

// DeleteEntry deletes a file or empty directory.
func DeleteEntry(rootPath, relativePath string) error {
	absPath, err := validatePath(rootPath, relativePath)
	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	if info.IsDir() {
		entries, _ := os.ReadDir(absPath)
		if len(entries) > 0 {
			return fmt.Errorf("directory not empty")
		}
		return os.Remove(absPath)
	}
	return os.Remove(absPath)
}

// RenameEntry renames a file or directory.
func RenameEntry(rootPath, oldPath, newPath string) error {
	absOld, err := validatePath(rootPath, oldPath)
	if err != nil {
		return err
	}
	absNew, err := validatePath(rootPath, newPath)
	if err != nil {
		return err
	}

	if _, err := os.Stat(absOld); err != nil {
		return fmt.Errorf("source: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(absNew), 0755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}

	return os.Rename(absOld, absNew)
}
