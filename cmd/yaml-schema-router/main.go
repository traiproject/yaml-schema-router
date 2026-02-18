// Package main is the entry point for k8s-yaml-router.
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"go.trai.ch/yaml-schema-router/internal/config"
	"go.trai.ch/yaml-schema-router/internal/detector"
	"go.trai.ch/yaml-schema-router/internal/detector/kubernetes"
	"go.trai.ch/yaml-schema-router/internal/lspproxy"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Fatal error: %v", err)
	}
}

func run() error {
	defaultLogPath := ""
	homeDir, err := os.UserHomeDir()
	if err == nil {
		defaultLogPath = filepath.Join(homeDir, ".config", config.DefaultConfigDirName, "router.log")
	} else {
		defaultLogPath = filepath.Join(os.TempDir(), "yaml-schema-router.log")
	}

	logFile := flag.String(
		"log-file",
		defaultLogPath,
		"Path to write logs (don't log to stdout!)",
	)
	lspPath := flag.String(
		"lsp-path",
		"yaml-language-server",
		"Path to the yaml-language-server executable. Defaults to checking the system PATH.",
	)
	flag.Parse()

	if *logFile != "" {
		logDir := filepath.Dir(*logFile)
		if err := os.MkdirAll(logDir, config.DefaultDirPerm); err != nil {
			return err
		}

		f, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, config.DefaultFilePerm)
		if err != nil {
			return err
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Printf("error closing file: %v", err)
			}
		}()
		log.SetOutput(f)
	} else {
		log.SetOutput(os.Stderr)
	}

	log.Printf("Starting yaml-schema-router. Using LSP executable: %s", *lspPath)

	k8sDetector := &kubernetes.K8sDetector{}
	chain := detector.NewChain(k8sDetector)

	proxy := lspproxy.NewProxy(*lspPath, chain)

	if err := proxy.Start(); err != nil {
		return err
	}

	log.Println("Proxy shut down cleanly.")

	return nil
}
