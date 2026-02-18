package kubernetes

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"go.trai.ch/yaml-schema-router/internal/config"
	"go.trai.ch/yaml-schema-router/internal/detector"
)

type schemaRef struct {
	Ref string `json:"$ref"`
}

type schemaProperties struct {
	Metadata schemaRef `json:"metadata"`
}

type schemaExtension struct {
	Properties schemaProperties `json:"properties"`
}

type schemaWrapper struct {
	AllOf []any `json:"allOf"`
}

// CRDDetector implements the detector.Detector interface for Kubernetes CRDs.
type CRDDetector struct{}

var _ detector.Detector = (*CRDDetector)(nil)

// Name returns the unique string identifier for the CRD detector.
func (d *CRDDetector) Name() string {
	return "kubernetes-crd"
}

// Detect inspects the YAML content for an apiVersion containing a custom group
// and constructs a wrapped JSON schema that includes standard ObjectMeta.
func (d *CRDDetector) Detect(_ string, content []byte) (schemaURL string, detected bool, err error) {
	apiVersion, kind := extractTypeMeta(content)

	if apiVersion == "" || kind == "" {
		return "", false, nil
	}

	group, version, found := strings.Cut(apiVersion, "/")
	if !found || (!strings.Contains(group, ".") || strings.HasSuffix(group, "k8s.io")) {
		return "", false, nil
	}

	kindFormatted := strings.ToLower(kind)
	fileName := fmt.Sprintf("%s_%s.json", kindFormatted, version)

	// Get the base CRD schema URL
	baseCRDURL, err := url.JoinPath(
		config.DefaultCRDSchemaRegistry,
		group,
		fileName,
	)
	if err != nil {
		return "", false, err
	}

	// Get the standard k8s ObjectMeta schema URL
	objectMetaURL, err := url.JoinPath(
		config.DefaultK8sSchemaRegistry,
		fmt.Sprintf("%s%s", config.DefaultK8sSchemaVersion, config.DefaultK8sSchemaFlavour),
		"objectmeta-meta-v1.json",
	)
	if err != nil {
		return "", false, err
	}

	fileURI, err := generateWrapperSchema(baseCRDURL, objectMetaURL, group, version, kindFormatted)
	if err != nil {
		return "", false, err
	}

	return fileURI, true, nil
}

// generateWrapperSchema builds the CRD wrapper and writes it to a temporary file.
func generateWrapperSchema(baseCRDURL, objectMetaURL, group, version, kindFormatted string) (string, error) {
	// Construct the wrapper schema using typed structs
	wrapper := schemaWrapper{
		AllOf: []any{
			schemaRef{Ref: baseCRDURL},
			schemaExtension{
				Properties: schemaProperties{
					Metadata: schemaRef{Ref: objectMetaURL},
				},
			},
		},
	}

	wrapperBytes, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return "", err
	}

	// Write the generated schema to a temporary file
	tempDir := filepath.Join(os.TempDir(), "yaml-schema-router-crds")
	if err := os.MkdirAll(tempDir, config.DefaultDirPerm); err != nil {
		return "", err
	}

	wrapperFileName := fmt.Sprintf("%s-%s-%s-wrapper.json", group, version, kindFormatted)
	wrapperFilePath := filepath.Join(tempDir, wrapperFileName)

	if err := os.WriteFile(wrapperFilePath, wrapperBytes, config.DefaultFilePerm); err != nil {
		return "", err
	}

	// Return the local file URI to the LSP
	return fmt.Sprintf("file://%s", wrapperFilePath), nil
}
