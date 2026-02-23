package schemaregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"go.trai.ch/yaml-schema-router/internal/config"
)

const componentName = "Registry"

// Registry manages a persistent disk cache for JSON schemas.
type Registry struct {
	baseDir string
}

type compositeSchema struct {
	AnyOf []map[string]string `json:"anyOf"`
}

// NewRegistry initializes the user's cache directory.
func NewRegistry() (*Registry, error) {
	userCache, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine user cache dir: %w", err)
	}

	baseDir := filepath.Join(userCache, config.DefaultConfigDirName, "schemas")
	if err := os.MkdirAll(baseDir, config.DefaultDirPerm); err != nil {
		return nil, fmt.Errorf("could not create cache dir: %w", err)
	}

	return &Registry{baseDir: baseDir}, nil
}

// GetSchemaURI checks if the schema exists on disk. If not, it attempts to
// download it. Returns a file:// URI on success, or an error if it fails.
func (r *Registry) GetSchemaURI(remoteURL, cachePath string) (string, error) {
	fullPath := filepath.Join(r.baseDir, cachePath)

	// Fast path: check if file already exists in cache
	if _, err := os.Stat(fullPath); err == nil {
		log.Printf("[%s] Cache hit: %s", componentName, cachePath)
		return fmt.Sprintf("file://%s", fullPath), nil
	}

	log.Printf("[%s] Cache miss: %s. Downloading from %s ...", componentName, cachePath, remoteURL)

	// Cache miss: download the schema
	data, err := download(remoteURL)
	if err != nil {
		// Return the error instead of falling back blindly
		return "", fmt.Errorf("failed to download %s: %w", remoteURL, err)
	}

	log.Printf("[%s] Download successful. Saving to %s", componentName, fullPath)

	// Save to cache
	if err := r.SaveLocalSchema(cachePath, data); err != nil {
		return "", fmt.Errorf("failed to save %s: %w", fullPath, err)
	}

	return fmt.Sprintf("file://%s", fullPath), nil
}

// GetLocalPath returns the absolute local path for a cache path.
func (r *Registry) GetLocalPath(cachePath string) string {
	return filepath.Join(r.baseDir, cachePath)
}

// SaveLocalSchema writes raw byte data directly to the cache. Useful for generated wrappers.
func (r *Registry) SaveLocalSchema(cachePath string, data []byte) error {
	fullPath := filepath.Join(r.baseDir, cachePath)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, config.DefaultDirPerm); err != nil {
		return err
	}

	return os.WriteFile(fullPath, data, config.DefaultFilePerm)
}

// GetLocalFileURI returns the formatted file:// URI for a known local cache path, without downloading.
func (r *Registry) GetLocalFileURI(cachePath string) string {
	fullPath := filepath.Join(r.baseDir, cachePath)
	return fmt.Sprintf("file://%s", fullPath)
}

// GenerateCompositeSchema creates a single schema using 'anyOf' to aggregate multiple schemas.
func (r *Registry) GenerateCompositeSchema(schemaURIs []string) (string, error) {
	if len(schemaURIs) == 0 {
		return "", nil
	}
	// Fast path: if there is only one manifest, just return its schema directly.
	if len(schemaURIs) == 1 {
		return schemaURIs[0], nil
	}

	// Deduplicate and sort URIs to ensure a deterministic hash for identical documents
	uriMap := make(map[string]bool)
	var uniqueURIs []string
	for _, uri := range schemaURIs {
		if !uriMap[uri] {
			uriMap[uri] = true
			uniqueURIs = append(uniqueURIs, uri)
		}
	}
	sort.Strings(uniqueURIs)

	// Hash the sorted URIs
	hash := sha256.New()
	for _, uri := range uniqueURIs {
		hash.Write([]byte(uri))
	}
	hashStr := hex.EncodeToString(hash.Sum(nil))[:16]

	cachePath := filepath.Join("composite", fmt.Sprintf("composite_%s.json", hashStr))

	// Fast path: check if this exact composite combination already exists
	if _, err := os.Stat(r.GetLocalPath(cachePath)); err == nil {
		return r.GetLocalFileURI(cachePath), nil
	}

	// Build the wrapper
	var anyOf []map[string]string
	for _, uri := range uniqueURIs {
		anyOf = append(anyOf, map[string]string{"$ref": uri})
	}

	composite := compositeSchema{AnyOf: anyOf}
	data, err := json.MarshalIndent(composite, "", "  ")
	if err != nil {
		return "", err
	}

	// Save the dynamically generated wrapper to disk
	if err := r.SaveLocalSchema(cachePath, data); err != nil {
		return "", err
	}

	return r.GetLocalFileURI(cachePath), nil
}
