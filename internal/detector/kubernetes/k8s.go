// Package kubernetes implements a schema detector for standard Kubernetes manifests.
package kubernetes

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/goccy/go-yaml"
	"go.trai.ch/yaml-schema-router/internal/config"
	"go.trai.ch/yaml-schema-router/internal/detector"
)

// k8sPeek is a minimal struct to extract only the necessary routing fields.
type k8sPeek struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

// K8sDetector implements the detector.Detector interface for Kubernetes manifests.
type K8sDetector struct{}

var _ detector.Detector = (*K8sDetector)(nil)

// Name returns the unique string identifier for the Kubernetes detector.
func (d *K8sDetector) Name() string {
	return "kubernetes-builtin"
}

// Detect inspects the YAML content for its Kubernetes apiVersion and kind to construct the appropriate schema URL.
func (d *K8sDetector) Detect(_ string, content []byte) (schemaURL string, detected bool, err error) {
	var peek k8sPeek
	if Unmarshalerr := yaml.Unmarshal(content, &peek); Unmarshalerr != nil {
		//nolint:nilerr // Intentional: if it fails to parse, it's simply not a valid K8s file.
		return "", false, nil
	}

	if peek.APIVersion == "" || peek.Kind == "" {
		return "", false, nil
	}

	// Example mapping logic (e.g., apps/v1, Deployment -> deployment-apps-v1.json)
	apiVersionFormatted := strings.ReplaceAll(peek.APIVersion, "/", "-")
	kindFormatted := strings.ToLower(peek.Kind)
	fileName := fmt.Sprintf("%s-%s.json", kindFormatted, apiVersionFormatted)

	versionDir := fmt.Sprintf("%s%s", config.DefaultK8sSchemaVersion, config.DefaultK8sSchemaFlavour)

	schemaURL, err = url.JoinPath(
		config.DefaultK8sSchemaRegistry,
		versionDir,
		fileName,
	)
	if err != nil {
		return "", false, err
	}

	return schemaURL, true, nil
}
