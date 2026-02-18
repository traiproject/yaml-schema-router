// Package config holds the application-wide configuration, default settings,
// and file permission constants for k8s-yaml-router.
package config

import "os"

const (
	// DefaultDirPerm represents standard directory permissions (rwxr-xr-x).
	DefaultDirPerm os.FileMode = 0o755

	// DefaultFilePerm represents standard file permissions (rw-rw-rw-).
	DefaultFilePerm os.FileMode = 0o666

	// DefaultConfigDirName is the folder name inside ~/.config/
	DefaultConfigDirName = "yaml-schema-router"
)
