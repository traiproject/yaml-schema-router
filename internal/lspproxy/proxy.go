// Package lspproxy implements a transparent proxy that intercepts and modifies
// LSP traffic between the editor and language server.
package lspproxy

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	"go.trai.ch/yaml-schema-router/internal/detector"
)

// Proxy manages the MITM connection between the Editor and the Language Server.
type Proxy struct {
	editorIn  io.Reader
	editorOut io.Writer

	serverCmd *exec.Cmd
	serverIn  io.WriteCloser
	serverOut io.ReadCloser

	lspPath       string
	detectorChain *detector.Chain

	// schemaState tracks URI -> applied Schema URL to prevent redundant updates
	schemaState map[string]string
	stateMutex  sync.RWMutex
}

// NewProxy initializes the structs and prepares the subprocess.
func NewProxy(lspPath string, chain *detector.Chain) *Proxy {
	return &Proxy{
		editorIn:      os.Stdin,
		editorOut:     os.Stdout,
		lspPath:       lspPath,
		detectorChain: chain,
		schemaState:   make(map[string]string),
	}
}

// Start launches the yaml-language-server and begins proxying traffic.
func (p *Proxy) Start() error {
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

	log.Printf("Language server started (PID: %d)", p.serverCmd.Process.Pid)

	var wg sync.WaitGroup

	wg.Go(func() {
		p.processServerToEditor()
	})

	wg.Go(func() {
		defer func() {
			if err := p.serverIn.Close(); err != nil {
				log.Fatalf("failed to close the IO reader: %v", err)
			}
		}()

		p.processEditorToServer()
	})

	err := p.serverCmd.Wait()

	wg.Wait()

	return err
}
