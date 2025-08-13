package util

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/mholt/archiver/v3"
)

var (
	ErrNotFullyCopied = errors.New("didn't copy entire file from the archive")
)

func ExtractArchive(archivePath string) (string, error) {
	// Our path will have a .temp appended to it so we can't rely on the automatic file-extension based archive extractor selection.
	// This code was taken from the archiver.Walk(archive string, walkFn WalkFunc) error function.
	// We just remove .temp before trying to find a matching archive extractor.
	wIface, err := archiver.ByExtension(archivePath[:len(archivePath)-len(".temp")])
	if err != nil {
		return "", err
	}
	w, ok := wIface.(archiver.Walker)
	if !ok {
		return "", fmt.Errorf("format specified by archive filename is not a walker format: %s (%T)", archivePath, wIface)
	}

	extractedFiles := []string{}
	baseDir := filepath.Dir(archivePath)
	
	err = w.Walk(archivePath, func(f archiver.File) error {
		// Skip directories
		if f.IsDir() {
			return nil
		}

		extractedPath := filepath.Join(baseDir, f.Name()+".temp")
		
		// Create directory structure if needed
		if err := os.MkdirAll(filepath.Dir(extractedPath), 0755); err != nil {
			return err
		}

		out, err := os.Create(extractedPath)
		if err != nil {
			return err
		}

		copied, err := io.Copy(out, f)
		if err != nil {
			out.Close()
			return err
		}
		if copied != f.Size() {
			out.Close()
			return ErrNotFullyCopied
		}

		err = out.Close()
		if err != nil {
			return err
		}

		extractedFiles = append(extractedFiles, extractedPath)
		return nil
	})

	if err != nil {
		// Clean up any partially extracted files on error
		for _, path := range extractedFiles {
			os.Remove(path)
		}
		return "", err
	}

	// Remove the original archive file
	err = os.Remove(archivePath)
	if err != nil {
		log.Println("remove error", err)
	}

	// If we extracted exactly one file, return that file path
	if len(extractedFiles) == 1 {
		return extractedFiles[0], nil
	} else if len(extractedFiles) > 1 {
		// Multiple files extracted, return the base directory
		return baseDir, nil
	} else {
		// No files extracted, this shouldn't normally happen for valid archives
		return archivePath, nil
	}
}

// IsArchive returns true if the file at the given path is an archive that can
// be extracted. Returns false otherwise.
func IsArchive(path string) bool {
	if filepath.Ext(path) == ".temp" {
		path = path[:len(path)-len(".temp")]
	}

	_, err := archiver.ByExtension(path)
	return err == nil
}
