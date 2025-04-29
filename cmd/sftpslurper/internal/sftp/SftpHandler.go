package sftp

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
)

/*
 * Handler implements all the required SFTP interfaces
 * for putting, listing, getting, and deleting files.
 */
type Handler struct {
	RootPath string
}

// Fileread implements sftp.FileReader
func (h *Handler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	log.Printf("Read request for: %s", r.Filepath)

	// Construct the full path for the file
	filePath := filepath.Join(h.RootPath, r.Filepath)

	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Failed to open file for reading: %v", err)
		return nil, err
	}

	return file, nil
}

// Filewrite implements sftp.FileWriter
func (h *Handler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	log.Printf("Write request for: %s", r.Filepath)

	// Create the upload directory if it doesn't exist
	if err := os.MkdirAll(h.RootPath, 0755); err != nil {
		return nil, err
	}

	// Construct the full path for the file
	filePath := filepath.Join(h.RootPath, r.Filepath)

	// Create the directory structure if it doesn't exist
	dirPath := filepath.Dir(filePath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, err
	}

	log.Printf("Writing file to: %s", filePath)

	// Create and return the file
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("Failed to open file for writing: %v", err)
		return nil, err
	}

	return file, nil
}

// Filecmd implements sftp.FileCmder
func (h *Handler) Filecmd(r *sftp.Request) error {
	log.Printf("Command request: %s on %s", r.Method, r.Filepath)

	// Construct the full path
	path := filepath.Join(h.RootPath, r.Filepath)

	switch r.Method {
	case "Setstat", "Setattr":
		// Handle file attribute changes (we'll just log it for now)
		log.Printf("Setstat/Setattr called on %s. This is not implemented", path)
		return nil

	case "Rename":
		// Handle rename operation
		// Todo: ensure we don't rename things outside the root directory
		oldPath := path
		newPath := filepath.Join(h.RootPath, r.Target)
		log.Printf("Renaming %s to %s", oldPath, newPath)
		return os.Rename(oldPath, newPath)

	case "Rmdir":
		// Handle remove directory.
		// Todo: ensure we don't remove the root directory or higher
		// as a security measure.
		log.Printf("Removing directory %s", path)
		return os.Remove(path)

	case "Mkdir":
		// Handle make directory
		// Todo: ensure we don't make things outside the root directory
		log.Printf("Creating directory %s", path)
		return os.MkdirAll(path, 0755)

	case "Remove", "Rm":
		// Handle remove file
		// Todo: ensure we don't remove things outside the root directory
		log.Printf("Removing file %s", path)
		return os.Remove(path)

	case "Symlink":
		// Handle symlink creation
		oldPath := path
		newPath := filepath.Join(h.RootPath, r.Target)
		log.Printf("Creating symlink from %s to %s", oldPath, newPath)
		return os.Symlink(oldPath, newPath)

	default:
		return fmt.Errorf("unsupported command method: %s", r.Method)
	}
}

// Filelist implements sftp.FileLister
func (h *Handler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	log.Printf("List request for: %s", r.Filepath)

	// Construct the full path
	path := filepath.Join(h.RootPath, r.Filepath)

	switch r.Method {
	case "List":
		// Get file info
		fi, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		if !fi.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", path)
		}

		// Read directory contents
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}

		// Convert to FileInfo slice
		fileInfos := make([]os.FileInfo, 0, len(entries))
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				log.Printf("Error getting info for %s: %v", entry.Name(), err)
				continue
			}
			fileInfos = append(fileInfos, info)
		}

		return ListerAt(fileInfos), nil

	case "Stat":
		// Get file info for a single file/directory
		fi, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		return ListerAt([]os.FileInfo{fi}), nil

	case "Readlink":
		// Read the target of a symlink
		target, err := os.Readlink(path)
		if err != nil {
			return nil, err
		}

		// Create a virtual file info for the symlink target
		linkInfo := virtualFileInfo{
			name:    filepath.Base(target),
			size:    0,
			mode:    os.ModeSymlink | 0644,
			modTime: time.Now(),
			isDir:   false,
		}

		return ListerAt([]os.FileInfo{linkInfo}), nil

	default:
		return nil, fmt.Errorf("unsupported list method: %s", r.Method)
	}
}

// virtualFileInfo implements os.FileInfo for virtual files
type virtualFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (v virtualFileInfo) Name() string       { return v.name }
func (v virtualFileInfo) Size() int64        { return v.size }
func (v virtualFileInfo) Mode() os.FileMode  { return v.mode }
func (v virtualFileInfo) ModTime() time.Time { return v.modTime }
func (v virtualFileInfo) IsDir() bool        { return v.isDir }
func (v virtualFileInfo) Sys() any           { return nil }
