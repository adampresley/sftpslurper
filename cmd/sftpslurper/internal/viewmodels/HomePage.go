package viewmodels

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/adampresley/sftpslurper/cmd/sftpslurper/internal/filetypes"
	"github.com/dustin/go-humanize"
)

type Home struct {
	BaseViewModel

	Files  []File
	Root   string
	Parent string
}

type File struct {
	Icon           string
	IsDirectory    bool
	CanBePreviewed bool
	Ext            string
	DirPath        string
	Name           template.HTML
	Date           string
	Size           string
}

func NewFileFromOS(f os.DirEntry, root string) (File, error) {
	var (
		err      error
		fileInfo os.FileInfo
	)

	ext := filepath.Ext(f.Name())
	result := File{
		Icon:           "icon " + getIcon(ext, f.IsDir()),
		IsDirectory:    f.IsDir(),
		Ext:            strings.TrimPrefix(ext, "."),
		CanBePreviewed: isPreviewable(ext),
		DirPath:        "",
		Name:           template.HTML(f.Name()),
	}

	if fileInfo, err = f.Info(); err != nil {
		return result, fmt.Errorf("failed to get file info: %w", err)
	}

	result.Date = fileInfo.ModTime().Format("2006-01-02 15:04:05")
	result.Size = humanize.Bytes(uint64(fileInfo.Size()))

	if result.IsDirectory {
		result.Size = ""

		if root == "" {
			result.DirPath = f.Name()
		} else {
			result.DirPath = filepath.Join(root, f.Name())
		}

		result.DirPath = filepath.ToSlash(result.DirPath)
	}

	return result, nil
}

func getIcon(ext string, isDir bool) string {
	if isDir {
		return "icon-folder"
	}

	if icon, ok := filetypes.IconMapping[ext]; ok {
		return icon
	}

	return "icon-file"
}

func isPreviewable(ext string) bool {
	previewable := map[string]struct{}{
		".txt":  {},
		".jpg":  {},
		".jpeg": {},
		".png":  {},
		".webp": {},
		".mp3":  {},
		".m4a":  {},
		".wav":  {},
		".mp4":  {},
		".mov":  {},
	}

	if _, ok := previewable[ext]; ok {
		return true
	}

	return false
}
