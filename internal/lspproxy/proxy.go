// Package lspproxy implements a transparent proxy that intercepts and modifies
// LSP traffic between the editor and language server.
package lspproxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	"go.trai.ch/yaml-schema-router/internal/detector"
	"go.trai.ch/yaml-schema-router/internal/schemaregistry"
)

const componentName = "Proxy"

// Proxy manages the MITM connection between the Editor and the Language Server.
type Proxy struct {
	editorIn  io.Reader
	editorOut io.Writer

	serverCmd *exec.Cmd
	serverIn  io.WriteCloser
	serverOut io.ReadCloser

	lspPath       string
	detectorChain *detector.Chain
	registry      *schemaregistry.Registry

	// schemaState tracks URI -> applied Schema URL to prevent redundant updates
	schemaState map[string]string
	stateMutex  sync.RWMutex
}

// NewProxy initializes the structs and prepares the subprocess.
func NewProxy(lspPath string, chain *detector.Chain, registry *schemaregistry.Registry) *Proxy {
	return &Proxy{
		editorIn:      os.Stdin,
		editorOut:     os.Stdout,
		lspPath:       lspPath,
		detectorChain: chain,
		registry:      registry,
		schemaState:   make(map[string]string),
	}
}

// Start launches the yaml-language-server and begins proxying traffic.
func (p *Proxy) Start(ctx context.Context) error {
	//nolint:gosec // lspPath is provided via a trusted command-line flag
	p.serverCmd = exec.Command(p.lspPath, "--stdio")

	serverIn, inErr := p.serverCmd.StdinPipe()
	if inErr != nil {
		return fmt.Errorf("failed to create stdin pipe to server: %w", inErr)
	}
	p.serverIn = serverIn

	serverOut, outErr := p.serverCmd.StdoutPipe()
	if outErr != nil {
		return fmt.Errorf("failed to create stdout pipe from server: %w", outErr)
	}
	p.serverOut = serverOut

	p.serverCmd.Stderr = os.Stderr

	if err := p.serverCmd.Start(); err != nil {
		return fmt.Errorf("failed to start language server (%s): %w", p.lspPath, err)
	}

	log.Printf("[%s] Language server started (PID: %d)", componentName, p.serverCmd.Process.Pid)

	var wg sync.WaitGroup

	wg.Go(func() {
		p.processServerToEditor()
	})

	wg.Go(func() {
		defer func() {
			if err := p.serverIn.Close(); err != nil {
				log.Printf("[%s] failed to close the IO reader: %v", componentName, err)
			}
		}()

		p.processEditorToServer()
	})

	cmdDone := make(chan error, 1)
	go func() {
		cmdDone <- p.serverCmd.Wait()
	}()

	// Block until either the process exits OR we receive a shutdown signal
	select {
	case err := <-cmdDone:
		// The language server exited on its own (or crashed)
		return err

	case <-ctx.Done():
		// The editor sent a signal (e.g., SIGTERM), so we shut down gracefully
		log.Printf("[%s] Context canceled, stopping language server...", componentName)

		// Kill the child process so we don't leave orphans
		if p.serverCmd.Process != nil {
			_ = p.serverCmd.Process.Kill()
		}

		return nil
	}
}
