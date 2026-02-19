// Package kubernetes implements a schema detector for standard Kubernetes manifests.
package kubernetes

import (
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"strings"

	"go.trai.ch/yaml-schema-router/internal/config"
	"go.trai.ch/yaml-schema-router/internal/detector"
	"go.trai.ch/yaml-schema-router/internal/schemaregistry"
)

// K8sDetector implements the detector.Detector interface for Kubernetes manifests.
type K8sDetector struct {
	Registry *schemaregistry.Registry
}

var _ detector.Detector = (*K8sDetector)(nil)

// K8sDetectorName is the unique identifier for the built-in Kubernetes detector.
const K8sDetectorName = "kubernetes-builtin"

// Name returns the unique string identifier for the Kubernetes detector.
func (d *K8sDetector) Name() string {
	return K8sDetectorName
}

// Detect inspects the YAML content for its Kubernetes apiVersion and kind to construct the appropriate schema URL.
func (d *K8sDetector) Detect(_ string, content []byte) (remoteSchemaURL string, detected bool, err error) {
	apiVersion, kind := extractTypeMeta(content)

	if apiVersion == "" || kind == "" {
		return "", false, nil
	}

	log.Printf("[%s] Found apiVersion='%s', kind='%s'", d.Name(), apiVersion, kind)

	// 1. Ignore the actual CustomResourceDefinition kind
	if kind == "CustomResourceDefinition" {
		log.Printf("[%s] Ignoring CustomResourceDefinition", d.Name())
		return "", false, nil
	}

	// 2. Ignore custom resources by analyzing the API group
	group := apiVersion
	if strings.Contains(group, "/") {
		group = strings.Split(group, "/")[0]
	}

	// Official Kubernetes API groups are either without dots (e.g., "apps", "batch")
	// or end with "k8s.io" (e.g., "rbac.authorization.k8s.io").
	// If the group contains a dot but doesn't end with k8s.io, it is a custom resource.
	if strings.Contains(group, ".") && !strings.HasSuffix(group, "k8s.io") {
		log.Printf("[%s] Ignoring Custom Resource (group: %s)", d.Name(), group)
		return "", false, nil
	}

	// Example mapping logic (e.g., apps/v1, Deployment -> deployment-apps-v1.json)
	apiVersionFormatted := strings.ReplaceAll(apiVersion, "/", "-")
	kindFormatted := strings.ToLower(kind)
	fileName := fmt.Sprintf("%s-%s.json", kindFormatted, apiVersionFormatted)

	versionDir := fmt.Sprintf("%s%s", config.DefaultK8sSchemaVersion, config.DefaultK8sSchemaFlavour)

	remoteSchemaURL, err = url.JoinPath(
		config.DefaultK8sSchemaRegistry,
		versionDir,
		fileName,
	)
	if err != nil {
		return "", false, err
	}

	log.Printf("[%s] Mapped to schema: %s", d.Name(), fileName)

	cachePath := filepath.Join(d.Name(), versionDir, fileName)
	localURI, err := d.Registry.GetSchemaURI(remoteSchemaURL, cachePath)
	if err != nil {
		return "", false, fmt.Errorf("failed to fetch schema: %w", err)
	}
	return localURI, true, nil
}

// extractTypeMeta scans the raw YAML content to quickly find the top-level
// apiVersion and kind.
func extractTypeMeta(content []byte) (apiVersion, kind string) {
	for line := range strings.SplitSeq(string(content), "\n") {
		// Only check top-level keys (no leading spaces)
		if after, ok := strings.CutPrefix(line, "apiVersion:"); ok {
			apiVersion = strings.TrimSpace(after)
			apiVersion = strings.Trim(apiVersion, `"'`)
		} else if after0, ok0 := strings.CutPrefix(line, "kind:"); ok0 {
			kind = strings.TrimSpace(after0)
			kind = strings.Trim(kind, `"'`)
		}

		if apiVersion != "" && kind != "" {
			break
		}
	}

	return apiVersion, kind
}
