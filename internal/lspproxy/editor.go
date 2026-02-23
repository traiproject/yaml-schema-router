package lspproxy

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"strings"
)

const (
	componentDidOpen      = "DidOpen"
	componentDidChange    = "DidChange"
	componentEditorServer = "Editor -> Server"
)

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

		p.handleEditorMessage(payload)
	}
}

func (p *Proxy) handleEditorMessage(payload []byte) {
	var msg BaseRPC
	if err := json.Unmarshal(payload, &msg); err != nil {
		p.forwardToServer(payload)
		return
	}

	if msg.Method != "" {
		if msg.Method == "textDocument/didOpen" ||
			msg.Method == "textDocument/didChange" ||
			msg.Method == "textDocument/didSave" {
			log.Printf("[%s] Intercepting method: %s", componentEditorServer, msg.Method)
		}

		switch msg.Method {
		case "textDocument/didOpen":
			p.handleDidOpen(payload)
		case "textDocument/didChange":
			p.handleDidChange(payload)
		}
		p.forwardToServer(payload)
		return
	}

	if msg.ID != nil && len(msg.Result) > 0 {
		payload = p.interceptWorkspaceConfiguration(&msg, payload)
	}

	p.forwardToServer(payload)
}

// triggerConfigurationPull sends an empty didChangeConfiguration notification
// to force the yaml-language-server to request a configuration pull.
func (p *Proxy) triggerConfigurationPull() {
	log.Printf("[%s] Triggering configuration pull (sending workspace/didChangeConfiguration)", componentName)
	// A barebones payload is enough to trigger the pullConfiguration() flow
	payload := []byte(`{"jsonrpc":"2.0","method":"workspace/didChangeConfiguration"}`)
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

	log.Printf("[%s] Processing file: %s", componentDidOpen, uri)

	if p.hasSchemaAnnotation(text) {
		log.Printf("[%s] Manual schema annotation detected for %s. Bypassing router.", componentDidOpen, uri)
		return
	}

	schemaURLs, err := p.detectorChain.Run(uri, []byte(text))
	if err != nil {
		log.Printf("[%s] Error running detectors: %v", componentDidOpen, err)
		return
	}

	if len(schemaURLs) == 0 {
		log.Printf("[%s] No schema detected for %s", componentDidOpen, uri)
		return
	}

	finalSchemaURL, err := p.registry.GenerateCompositeSchema(schemaURLs)
	if err != nil {
		log.Printf("[%s] Error generating composite schema: %v", componentDidOpen, err)
		return
	}

	log.Printf("[%s] MATCH! Mapping %s -> %s", componentDidOpen, uri, finalSchemaURL)

	p.stateMutex.Lock()
	p.schemaState[uri] = finalSchemaURL
	p.stateMutex.Unlock()

	p.triggerConfigurationPull()
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

	if p.hasSchemaAnnotation(text) {
		p.stateMutex.Lock()
		if _, exists := p.schemaState[uri]; exists {
			log.Printf("[%s] Manual schema annotation added to %s. Removing from router state.", componentDidChange, uri)
			delete(p.schemaState, uri)
			p.stateMutex.Unlock()

			p.triggerConfigurationPull()
		} else {
			p.stateMutex.Unlock()
		}
		return
	}

	if len(strings.TrimSpace(text)) == 0 {
		p.stateMutex.Lock()
		if _, exists := p.schemaState[uri]; exists {
			log.Printf("[%s] File content cleared for %s. Removing from router state.", componentDidChange, uri)
			delete(p.schemaState, uri)
			p.stateMutex.Unlock()

			p.triggerConfigurationPull()
		} else {
			p.stateMutex.Unlock()
		}
		return
	}

	schemaURLs, err := p.detectorChain.Run(uri, []byte(text))
	if err != nil {
		log.Printf("[%s] Error running detectors: %v", componentDidChange, err)
		return
	}

	if len(schemaURLs) == 0 {
		// TODO: If we lose detection (e.g. user deletes the apiVersion line), strictly we might want to
		// remove it from state, but for now we just return.
		return
	}

	finalSchemaURL, err := p.registry.GenerateCompositeSchema(schemaURLs)
	if err != nil {
		log.Printf("[%s] Error generating composite schema: %v", componentDidChange, err)
		return
	}

	p.stateMutex.Lock()
	// Only trigger a configuration pull if the schema actually changed
	if p.schemaState[uri] != finalSchemaURL {
		log.Printf("[%s] Schema changed for %s! New: %s", componentDidChange, uri, finalSchemaURL)
		p.schemaState[uri] = finalSchemaURL
		p.stateMutex.Unlock()

		p.triggerConfigurationPull()
	} else {
		p.stateMutex.Unlock()
	}
}
