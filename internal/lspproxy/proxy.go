// Package lspproxy implements a transparent proxy that intercepts and modifies
// LSP traffic between the editor and language server.
package lspproxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
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

// readLSPMessage reads the HTTP-like headers to find the Content-Length,
// then reads and returns the exact JSON payload.
func readLSPMessage(reader *bufio.Reader) ([]byte, error) {
	var contentLength int

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err // Usually io.EOF when the connection closes
		}

		line = strings.TrimSpace(line)

		// An empty line marks the end of the headers
		if line == "" {
			break
		}

		if strings.HasPrefix(line, "Content-Length:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				contentLength, err = strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("invalid Content-Length value '%s': %w", val, err)
				}
			}
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing or zero Content-Length header")
	}

	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, fmt.Errorf("failed to read full payload of size %d: %w", contentLength, err)
	}

	return payload, nil
}

// processEditorToServer continuously reads from the editor, parses headers,
// extracts the payload, and routes it based on the JSON-RPC method.
func (p *Proxy) processEditorToServer() {
	reader := bufio.NewReader(p.editorIn)

	for {
		payload, err := readLSPMessage(reader)
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Fatalf("Fatal error reading header from editor: %v", err)
		}

		var msg BaseRPC
		if err := json.Unmarshal(payload, &msg); err != nil {
			p.forwardToServer(payload)
			continue
		}

		if msg.Method != "" {
			switch msg.Method {
			case "textDocument/didOpen":
				p.handleDidOpen(payload)
			case "textDocument/didChange":
				p.handleDidChange(payload)
			}
			p.forwardToServer(payload)
			continue
		}

		// Intercept Helix's responses to `workspace/configuration`
		if msg.ID != nil && len(msg.Result) > 0 {
			payload = p.interceptWorkspaceConfiguration(msg, payload)
		}

		p.forwardToServer(payload)
	}
}

// processServerToEditor continuously reads from the language server,
// intercepts the initialize response to force Full Sync, and forwards to the editor.
func (p *Proxy) processServerToEditor() {
	reader := bufio.NewReader(p.serverOut)

	for {

		payload, err := readLSPMessage(reader)
		if err != nil {
			if err == io.EOF {
				// The server closed the connection
				return
			}
			log.Fatalf("Fatal error reading header from server: %v", err)
		}

		// Intercept and optionally rewrite the payload
		modifiedPayload := p.forceFullSync(payload)

		header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(modifiedPayload))
		if _, err := p.editorOut.Write([]byte(header)); err != nil {
			log.Printf("Error writing header to editor: %v", err)
			return
		}

		if _, err := p.editorOut.Write(modifiedPayload); err != nil {
			log.Printf("Error writing payload to editor: %v", err)
			return
		}
	}
}

// interceptWorkspaceConfiguration dynamically injects schema configurations
// into the editor's response to the language server.
func (p *Proxy) interceptWorkspaceConfiguration(msg BaseRPC, payload []byte) []byte {
	var result []any
	if err := json.Unmarshal(msg.Result, &result); err != nil || len(result) == 0 {
		return payload
	}

	// The first item in the array is the "yaml" section requested by the server
	yamlConfig, ok := result[0].(map[string]any)
	if !ok {
		return payload
	}

	p.stateMutex.RLock()
	groupedSchemas := make(map[string][]string)
	for uri, schemaURL := range p.schemaState {
		groupedSchemas[schemaURL] = append(groupedSchemas[schemaURL], uri)
	}
	p.stateMutex.RUnlock()

	// If no schemas are detected yet, return unmodified payload
	if len(groupedSchemas) == 0 {
		return payload
	}

	// Inject our schemas into Helix's response
	yamlConfig["schemas"] = groupedSchemas
	result[0] = yamlConfig

	newResult, err := json.Marshal(result)
	if err != nil {
		return payload
	}
	msg.Result = newResult

	modifiedPayload, err := json.Marshal(msg)
	if err != nil {
		return payload
	}

	return modifiedPayload
}

// forwardToServer cleanly re-serializes the header and sends the exact payload to the language server.
func (p *Proxy) forwardToServer(payload []byte) {
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))

	if _, err := p.serverIn.Write([]byte(header)); err != nil {
		log.Printf("Error writing header to server: %v", err)
		return
	}

	if _, err := p.serverIn.Write(payload); err != nil {
		log.Printf("Error writing payload to server: %v", err)
	}
}

func (p *Proxy) injectSchemaConfiguration() {
	p.stateMutex.RLock()
	// yaml-language-server expects: "schema-url": ["uri1", "uri2"]
	groupedSchemas := make(map[string][]string)
	for uri, schemaURL := range p.schemaState {
		groupedSchemas[schemaURL] = append(groupedSchemas[schemaURL], uri)
	}
	p.stateMutex.RUnlock()

	notification := DidChangeConfigurationNotification{
		JSONRPC: "2.0",
		Method:  "workspace/didChangeConfiguration",
		Params: DidChangeConfigurationParams{
			Settings: Settings{
				YAML: YAMLSound{
					Schemas: groupedSchemas,
				},
			},
		},
	}

	payload, err := json.Marshal(notification)
	if err != nil {
		log.Printf("Failed to marshal config notification: %v", err)
		return
	}

	p.forwardToServer(payload)
}

func (p *Proxy) handleDidOpen(payload []byte) {
	var notif DidOpenNotification
	if err := json.Unmarshal(payload, &notif); err != nil {
		log.Printf("Error unmarshaling didOpen: %v", err)
		return
	}

	uri := notif.Params.TextDocument.URI
	text := notif.Params.TextDocument.Text

	schemaURL, detected, err := p.detectorChain.Run(uri, []byte(text))
	if err != nil || !detected {
		return
	}

	p.stateMutex.Lock()
	p.schemaState[uri] = schemaURL
	p.stateMutex.Unlock()

	p.injectSchemaConfiguration()
}

func (p *Proxy) handleDidChange(payload []byte) {
	var notif DidChangeNotification
	if err := json.Unmarshal(payload, &notif); err != nil {
		log.Printf("Error unmarshaling didChange: %v", err)
		return
	}

	if len(notif.Params.ContentChanges) == 0 {
		return
	}

	uri := notif.Params.TextDocument.URI
	text := notif.Params.ContentChanges[0].Text

	schemaURL, detected, err := p.detectorChain.Run(uri, []byte(text))
	if err != nil || !detected {
		return
	}

	p.stateMutex.Lock()
	// Only inject if the schema actually changed
	if p.schemaState[uri] != schemaURL {
		p.schemaState[uri] = schemaURL
		p.stateMutex.Unlock()
		p.injectSchemaConfiguration()
	} else {
		p.stateMutex.Unlock()
	}
}

// forceFullSync intercepts the 'initialize' response from the language server
// and overwrites the textDocumentSync capability to 1 (Full Sync).
func (p *Proxy) forceFullSync(payload []byte) []byte {
	// Fast path: avoid JSON unmarshaling if this clearly isn't an initialize response.
	// We just look for "capabilities" and "textDocumentSync".
	payloadStr := string(payload)
	if !strings.Contains(payloadStr, `"capabilities"`) || !strings.Contains(payloadStr, `"textDocumentSync"`) {
		return payload
	}

	var msg BaseRPC
	if err := json.Unmarshal(payload, &msg); err != nil || len(msg.Result) == 0 {
		return payload
	}

	var result map[string]any
	if err := json.Unmarshal(msg.Result, &result); err != nil {
		return payload
	}

	capabilities, ok := result["capabilities"].(map[string]any)
	if !ok {
		return payload
	}

	if _, exists := capabilities["textDocumentSync"]; exists {
		capabilities["textDocumentSync"] = 1
		result["capabilities"] = capabilities

		if newResult, resultErr := json.Marshal(result); resultErr == nil {
			msg.Result = newResult
			if newPayload, msgErr := json.Marshal(msg); msgErr == nil {
				log.Println("Successfully intercepted 'initialize' and forced textDocumentSync to Full (1)")
				return newPayload
			}
			log.Printf("Warning: failed to re-marshal modified capabilities: %v", resultErr)
		}
	}

	return payload
}
