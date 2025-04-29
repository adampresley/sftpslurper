package configuration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/app-nerds/configinator"
)

type Config struct {
	LogLevel string `flag:"loglevel" env:"LOG_LEVEL" default:"debug" description:"The log level to use. Valid values are 'debug', 'info', 'warn', and 'error'"`
	Host     string `flag:"h" env:"HOST" default:"localhost:8080" description:"The address and port to bind the HTTP server to"`
	SftpHost string `flag:"sftph" env:"SFTP_HOST" default:"localhost:2200" description:"Address to listen on for the SFTP server"`
}

func LoadConfig() Config {
	config := Config{}
	configinator.Behold(&config)
	return config
}

// SanitizePath ensures that a given path cannot traverse outside the upload folder.
// It returns a safe, absolute path within the upload folder, or an empty string if the path
// would escape the upload folder boundary.
func (c *Config) SanitizePath(requestedPath string) (string, error) {
	uploadFolderAbs, _ := filepath.Abs(UploadFolder)

	// Ensure the upload folder exists
	if _, err := os.Stat(uploadFolderAbs); os.IsNotExist(err) {
		if err := os.MkdirAll(uploadFolderAbs, 0755); err != nil {
			return "", err
		}
	}

	// Join the requested path with the upload folder
	// This handles both absolute and relative paths
	targetPath := filepath.Join(uploadFolderAbs, filepath.Clean(requestedPath))

	// Clean the path to resolve any ".." or "." components
	targetPath = filepath.Clean(targetPath)

	// Ensure the target path is still within the upload folder
	if !strings.HasPrefix(targetPath, uploadFolderAbs) {
		return "", fmt.Errorf("Invalid path traversal attempt: %s", requestedPath)
	}

	return targetPath, nil
}
