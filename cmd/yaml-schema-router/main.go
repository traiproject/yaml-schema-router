// Package main is the entry point for k8s-yaml-router.
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"go.trai.ch/yaml-schema-router/internal/config"
)

func main() {
	defaultLogPath := ""
	homeDir, err := os.UserHomeDir()
	if err == nil {
		defaultLogPath = filepath.Join(homeDir, ".config", "k8s-yaml-router", "router.log")
	} else {
		defaultLogPath = filepath.Join(os.TempDir(), "k8s-yaml-router.log")
	}

	logFile := flag.String("log-file", defaultLogPath, "Path to write logs (don't log to stdout!)")
	flag.Parse()

	if *logFile != "" {
		logDir := filepath.Dir(*logFile)
		if err := os.MkdirAll(logDir, config.DefaultDirPerm); err != nil {
			log.Fatalf("failed to create log directory: %v", err)
		}

		f, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, config.DefaultFilePerm)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Fatalf("error closing file: %v", err)
			}
		}()
		log.SetOutput(f)
	} else {
		log.SetOutput(os.Stderr)
	}
}
