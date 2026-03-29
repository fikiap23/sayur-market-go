package storage

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"product-service/config"
	"strings"

	"github.com/labstack/gommon/log"
	storage_go "github.com/supabase-community/storage-go"
)

type SupabaseInterface interface {
	UploadFile(path string, file io.Reader) (string, error)
}

type supabaseStruct struct {
	cfg *config.Config
}

// UploadFile implements SupabaseInterface.
// Uses a direct POST with GetBody and HTTP/1.1 so the Go client can retry safely;
// storage-go's UploadFile wraps the body in bufio without GetBody, which breaks on HTTP/2 retries.
func (s *supabaseStruct) UploadFile(path string, file io.Reader) (string, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("read upload body: %w", err)
	}

	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	base := strings.TrimSuffix(s.cfg.Storage.URL, "/")
	objectPath := removeDoubleSlash(s.cfg.Storage.Bucket + "/" + path)
	uploadURL := base + "/object/" + objectPath

	req, err := http.NewRequest(http.MethodPost, uploadURL, nil)
	if err != nil {
		return "", fmt.Errorf("build upload request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.Storage.Key)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("apikey", s.cfg.Storage.Key)

	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	req.Body = io.NopCloser(bytes.NewReader(data))
	req.ContentLength = int64(len(data))

	httpClient := &http.Client{
		Transport: &http.Transport{
			// Avoid HTTP/2 edge cases (e.g. PROTOCOL_ERROR + retry) with non-rewindable bodies in older clients.
			TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{},
		},
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Errorf("Error uploading file to bucket=%s path=%s: %v", s.cfg.Storage.Bucket, path, err)
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read upload response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		_ = json.Unmarshal(bodyBytes, &apiErr)
		msg := apiErr.Message
		if msg == "" {
			msg = apiErr.Error
		}
		if msg == "" {
			msg = strings.TrimSpace(string(bodyBytes))
		}
		log.Errorf("Error uploading file to bucket=%s path=%s: %s", s.cfg.Storage.Bucket, path, msg)
		return "", fmt.Errorf("failed to upload file: %s", msg)
	}

	client := storage_go.NewClient(s.cfg.Storage.URL, s.cfg.Storage.Key, map[string]string{"Content-Type": contentType})
	result := client.GetPublicUrl(s.cfg.Storage.Bucket, path)

	if result.SignedURL == "" {
		log.Errorf("Upload returned empty key for bucket=%s path=%s", s.cfg.Storage.Bucket, path)
	}

	return result.SignedURL, nil
}

func removeDoubleSlash(p string) string {
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	return p
}

func NewSupabase(cfg *config.Config) SupabaseInterface {
	return &supabaseStruct{cfg: cfg}
}
