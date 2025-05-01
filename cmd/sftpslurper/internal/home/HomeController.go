package home

import (
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/adampresley/adamgokit/httphelpers"
	"github.com/adampresley/adamgokit/rendering"
	"github.com/adampresley/sftpslurper/cmd/sftpslurper/internal/configuration"
	"github.com/adampresley/sftpslurper/cmd/sftpslurper/internal/viewmodels"
)

type HomeHandlers interface {
	HomePage(w http.ResponseWriter, r *http.Request)
	AboutPage(w http.ResponseWriter, r *http.Request)
	PreviewContent(w http.ResponseWriter, r *http.Request)
	ServeFile(w http.ResponseWriter, r *http.Request)
	DeleteFile(w http.ResponseWriter, r *http.Request)
}

type HomeControllerConfig struct {
	Config   *configuration.Config
	Renderer rendering.TemplateRenderer
}

type HomeController struct {
	config   *configuration.Config
	renderer rendering.TemplateRenderer
}

func NewHomeController(config HomeControllerConfig) HomeController {
	return HomeController{
		config:   config.Config,
		renderer: config.Renderer,
	}
}

func (c HomeController) HomePage(w http.ResponseWriter, r *http.Request) {
	var (
		err       error
		cleanRoot string
		osFiles   []os.DirEntry
	)

	pageName := "pages/home"

	viewData := viewmodels.Home{
		BaseViewModel: viewmodels.BaseViewModel{
			Version: c.config.Version,
			Message: "",
			IsHtmx:  httphelpers.IsHtmx(r),
			JavascriptIncludes: []rendering.JavascriptInclude{
				{
					Type: "module",
					Src:  "/static/js/pages/home.js",
				},
			},
		},
		Root:   strings.TrimSpace(httphelpers.GetFromRequest[string](r, "root")),
		Files:  []viewmodels.File{},
		Parent: "",
	}

	if cleanRoot, err = c.config.SanitizePath(viewData.Root); err != nil {
		slog.Error("error determining root path", "error", err, "root", viewData.Root)
		viewData.Message = "Invalid root path"
		viewData.IsError = true

		c.renderer.Render(pageName, viewData, w)
		return
	}

	if osFiles, err = os.ReadDir(cleanRoot); err != nil {
		slog.Error("error reading directory", "error", err, "root", cleanRoot)
		viewData.Message = "Unexpected error reading directory contents"
		viewData.IsError = true

		c.renderer.Render(pageName, viewData, w)
		return
	}

	if viewData.Root != "" {
		pathParts := strings.Split(filepath.ToSlash(viewData.Root), "/")

		if len(pathParts) > 1 {
			viewData.Parent = strings.Join(pathParts[:len(pathParts)-1], "/")
		}
	}

	for _, f := range osFiles {
		newFile, err := viewmodels.NewFileFromOS(f, viewData.Root)

		if err != nil {
			slog.Error("error reading file info", "error", err, "root", cleanRoot, "file", f.Name())
			viewData.Message = "Unexpected error reading file information"
			viewData.IsError = true

			c.renderer.Render(pageName, viewData, w)
			return
		}

		viewData.Files = append(viewData.Files, newFile)
	}

	slog.Info("rendering home page", "root", cleanRoot)
	c.renderer.Render(pageName, viewData, w)
}

/*
GET /uploads?path={path}
*/
func (c HomeController) ServeFile(w http.ResponseWriter, r *http.Request) {
	filePath := strings.TrimSpace(httphelpers.GetFromRequest[string](r, "path"))

	slog.Info("serving file", "path", filePath)

	if filePath == "" {
		slog.Error("no file path provided")
		http.Error(w, "No file path provided", http.StatusBadRequest)
		return
	}

	// Sanitize and validate the path
	cleanPath, err := c.config.SanitizePath(filePath)
	if err != nil {
		slog.Error("invalid file path", "error", err, "path", filePath)
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	// Check if file exists
	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Error("file not found", "path", cleanPath)
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		slog.Error("error accessing file", "error", err, "path", cleanPath)
		http.Error(w, "Error accessing file", http.StatusInternalServerError)
		return
	}

	// Ensure it's a file, not a directory
	if fileInfo.IsDir() {
		slog.Error("cannot serve a directory", "path", cleanPath)
		http.Error(w, "Cannot download a directory", http.StatusBadRequest)
		return
	}

	// Open the file
	file, err := os.Open(cleanPath)
	if err != nil {
		slog.Error("error opening file", "error", err, "path", cleanPath)
		http.Error(w, "Error opening file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set content disposition header for download
	fileName := filepath.Base(cleanPath)
	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)

	// Set content type based on file extension
	contentType := http.DetectContentType(make([]byte, 512)) // Default
	ext := strings.ToLower(filepath.Ext(fileName))
	if mimeType := mime.TypeByExtension(ext); mimeType != "" {
		contentType = mimeType
	}
	w.Header().Set("Content-Type", contentType)

	// Set content length
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Log the download
	slog.Info("serving file", "path", cleanPath, "size", fileInfo.Size())

	// Serve the file
	http.ServeContent(w, r, fileName, fileInfo.ModTime(), file)
}

/*
GET /preview?ext={ext}&filename={filename}&root={root}
*/
func (c HomeController) PreviewContent(w http.ResponseWriter, r *http.Request) {
	ext := strings.ToLower(httphelpers.GetFromRequest[string](r, "ext"))
	fileName := httphelpers.GetFromRequest[string](r, "filename")
	root := strings.TrimSpace(httphelpers.GetFromRequest[string](r, "root"))

	markup := ""

	switch ext {
	case "png", "jpeg", "jpg", "webp":
		markup = fmt.Sprintf(`<img src="/uploads?path=%s/%s" alt="%s" />`, root, fileName, fileName)
	case "m3a", "mp3", "wav", "ogg", "oga", "flac":
		markup = fmt.Sprintf(`<audio controls><source src="/uploads?path=%s/%s" type="%s" /></audio>`, root, fileName, ext)
	case "csv", "tsv", "pdf", "xls", "xlsx", "doc", "docx":
		c.ServeFile(w, r)
	case "txt":
		p := filepath.Join(root, fileName)
		p, _ = c.config.SanitizePath(p)
		slog.Info("rendering text preview", "path", p)
		textContent, err := os.ReadFile(p)

		if err != nil {
			slog.Error("error reading file", "error", err, "path", root, "file", fileName)
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}

		markup = fmt.Sprintf(`<pre>%s</pre>`, strings.ReplaceAll(string(textContent), "\n", "<br>"))
	default:
		c.ServeFile(w, r)
	}

	httphelpers.TextOK(w, markup)
}

/*
DELETE /uploads?root={root}&filename={filename}&isdir={isdir}
*/
func (c HomeController) DeleteFile(w http.ResponseWriter, r *http.Request) {
	root := strings.TrimSpace(httphelpers.GetFromRequest[string](r, "root"))
	filename := httphelpers.GetFromRequest[string](r, "name")
	isdir := httphelpers.GetFromRequest[bool](r, "isdir")

	// Construct the full path
	fullPath := filepath.Join(root, filename)

	// Sanitize and validate the path
	cleanPath, err := c.config.SanitizePath(fullPath)
	if err != nil {
		slog.Error("invalid file path for deletion", "error", err, "path", fullPath)
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	slog.Info("attempting to delete", "path", cleanPath, "isDirectory", isdir, "fullpath", fullPath)

	// Check if file/directory exists
	_, err = os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Error("file or directory not found for deletion", "path", cleanPath)
			http.Error(w, "File or directory not found", http.StatusNotFound)
			return
		}

		slog.Error("error accessing file or directory for deletion", "error", err, "path", cleanPath)
		http.Error(w, "Error accessing file or directory", http.StatusInternalServerError)
		return
	}

	// Delete the file or directory
	var deleteErr error
	if isdir {
		slog.Info("deleting directory", "path", cleanPath)
		deleteErr = os.RemoveAll(cleanPath)
	} else {
		slog.Info("deleting file", "path", cleanPath)
		deleteErr = os.Remove(cleanPath)
	}

	if deleteErr != nil {
		slog.Error("error deleting file or directory", "error", deleteErr, "path", cleanPath, "isDirectory", isdir)
		http.Error(w, fmt.Sprintf("Error deleting %s: %v", filename, deleteErr), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	httphelpers.TextOK(w, fmt.Sprintf("Successfully deleted %s", filename))
}

func (c HomeController) AboutPage(w http.ResponseWriter, r *http.Request) {
	pageName := "pages/about"

	viewData := viewmodels.AboutPage{
		BaseViewModel: viewmodels.BaseViewModel{
			Version:            c.config.Version,
			Message:            "",
			IsHtmx:             httphelpers.IsHtmx(r),
			JavascriptIncludes: []rendering.JavascriptInclude{},
		},
	}

	c.renderer.Render(pageName, viewData, w)
}
