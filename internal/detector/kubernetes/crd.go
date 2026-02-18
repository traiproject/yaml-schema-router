package kubernetes

import (
	"fmt"
	"net/url"
	"strings"

	"go.trai.ch/yaml-schema-router/internal/config"
	"go.trai.ch/yaml-schema-router/internal/detector"
)

// CRDDetector implements the detector.Detector interface for Kubernetes CRDs.
type CRDDetector struct{}

var _ detector.Detector = (*CRDDetector)(nil)

// Name returns the unique string identifier for the CRD detector.
func (d *CRDDetector) Name() string {
	return "kubernetes-crd-datree"
}

// Detect inspects the YAML content for an apiVersion containing a custom group
// and constructs the appropriate datreeio CRDs catalog URL.
func (d *CRDDetector) Detect(_ string, content []byte) (schemaURL string, detected bool, err error) {
	apiVersion, kind := extractTypeMeta(content)

	if apiVersion == "" || kind == "" {
		return "", false, nil
	}

	// Split the apiVersion into group and version (e.g., "cilium.io/v2" -> "cilium.io", "v2")
	group, version, found := strings.Cut(apiVersion, "/")
	if !found {
		// Native core API resources (like "v1") don't have a slash.
		return "", false, nil
	}

	// Filter for custom resources:
	// They typically have a dot in their group and don't end with "k8s.io"
	// (Native groups are like "apps", "batch", or "rbac.authorization.k8s.io")
	if !strings.Contains(group, ".") || strings.HasSuffix(group, "k8s.io") {
		return "", false, nil
	}

	kindFormatted := strings.ToLower(kind)
	fileName := fmt.Sprintf("%s_%s.json", kindFormatted, version)

	schemaURL, err = url.JoinPath(
		config.DefaultCRDSchemaRegistry,
		group,
		fileName,
	)
	if err != nil {
		return "", false, err
	}

	return schemaURL, true, nil
}
