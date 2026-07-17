package file

import (
	"testing"
	"time"

	"github.com/stashapp/stash/pkg/models"
)

func TestHasFileChangedDetectsSamePathReplacementWithDifferentSize(t *testing.T) {
	modTime := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	existing := &models.BaseFile{
		DirEntry: models.DirEntry{ModTime: modTime},
		Path:     "Test/image00001.jpg",
		Basename: "image00001.jpg",
		Size:     100,
	}
	scanned := ScannedFile{
		BaseFile: &models.BaseFile{
			DirEntry: models.DirEntry{ModTime: modTime},
			Path:     "Test/image00001.jpg",
			Basename: "image00001.jpg",
			Size:     200,
		},
	}

	if !hasFileChanged(scanned, existing) {
		t.Fatal("expected file replacement with the same path and mod time but different size to be treated as changed")
	}
}

func TestHasFileChangedIgnoresUnchangedFile(t *testing.T) {
	modTime := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	existing := &models.BaseFile{
		DirEntry: models.DirEntry{ModTime: modTime},
		Path:     "Test/image00001.jpg",
		Basename: "image00001.jpg",
		Size:     100,
	}
	scanned := ScannedFile{
		BaseFile: &models.BaseFile{
			DirEntry: models.DirEntry{ModTime: modTime},
			Path:     "Test/image00001.jpg",
			Basename: "image00001.jpg",
			Size:     100,
		},
	}

	if hasFileChanged(scanned, existing) {
		t.Fatal("expected identical scan metadata to be treated as unchanged")
	}
}
