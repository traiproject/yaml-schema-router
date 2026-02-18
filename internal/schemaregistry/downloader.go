// Package schemaregistry handles downloading and caching of JSON schemas.
package schemaregistry

import (
	"fmt"
	"io"
	"net/http"

	"go.trai.ch/yaml-schema-router/internal/config"
)

// download fetches the raw bytes from a given URL with a strict timeout.
func download(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: config.DefaultDownloaderTimeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
