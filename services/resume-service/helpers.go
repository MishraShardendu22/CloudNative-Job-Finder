package main

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

func sanitizeFilename(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "resume" + filepath.Ext(name)
	}
	return filenamePattern.ReplaceAllString(trimmed, "_")
}

func getResumeFile(r *http.Request) (multipart.File, *multipart.FileHeader, error) {
	file, header, err := r.FormFile("resume")
	if err == nil {
		return file, header, nil
	}
	file, header, err = r.FormFile("file")
	if err == nil {
		return file, header, nil
	}
	return nil, nil, err
}

func decodeKeywordJSON(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var output []string
	if err := json.Unmarshal(raw, &output); err != nil {
		return []string{}
	}
	return output
}
