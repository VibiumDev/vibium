package proxy

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// setupDownloads creates a temp dir and tells the browser to save downloads there.
func (r *Router) setupDownloads(session *BrowserSession) {
	dir, err := os.MkdirTemp("", "vibium-downloads-*")
	if err != nil {
		fmt.Printf("[router] Failed to create download temp dir: %v\n", err)
		return
	}

	session.mu.Lock()
	session.downloadDir = dir
	session.mu.Unlock()

	_, err = r.sendInternalCommand(session, "browser.setDownloadBehavior", map[string]interface{}{
		"downloadBehavior": map[string]interface{}{
			"type":              "allowed",
			"destinationFolder": dir,
		},
	})
	if err != nil {
		fmt.Printf("[router] Failed to set download behavior: %v\n", err)
	}
}

// handleDownloadSaveAs copies a downloaded file from the temp dir to a user-specified path.
func (r *Router) handleDownloadSaveAs(session *BrowserSession, cmd bidiCommand) {
	sourcePath, _ := cmd.Params["sourcePath"].(string)
	destPath, _ := cmd.Params["destPath"].(string)

	if sourcePath == "" || destPath == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("download.saveAs requires sourcePath and destPath"))
		return
	}

	// Validate that the source is within the download dir (prevent path traversal)
	session.mu.Lock()
	dlDir := session.downloadDir
	session.mu.Unlock()

	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("invalid source path: %w", err))
		return
	}

	absDlDir, _ := filepath.Abs(dlDir)
	if !strings.HasPrefix(absSource, absDlDir+string(filepath.Separator)) && absSource != absDlDir {
		r.sendError(session, cmd.ID, fmt.Errorf("source path is not within download directory"))
		return
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to create destination directory: %w", err))
		return
	}

	// Copy the file
	src, err := os.Open(absSource)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to open downloaded file: %w", err))
		return
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to create destination file: %w", err))
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to copy file: %w", err))
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"saved": true})
}
