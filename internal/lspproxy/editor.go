package lspproxy

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
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
			payload = p.interceptWorkspaceConfiguration(&msg, payload)
		}

		p.forwardToServer(payload)
	}
}

// triggerConfigurationPull sends an empty didChangeConfiguration notification
// to force the yaml-language-server to request a configuration pull.
func (p *Proxy) triggerConfigurationPull() {
	// A barebones payload is enough to trigger the pullConfiguration() flow
	payload := []byte(`{"jsonrpc":"2.0","method":"workspace/didChangeConfiguration"`)
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

	schemaURL, detected, err := p.detectorChain.Run(uri, []byte(text))
	if err != nil || !detected {
		return
	}

	p.stateMutex.Lock()
	// Only trigger a configuration pull if the schema actually changed
	if p.schemaState[uri] != schemaURL {
		p.schemaState[uri] = schemaURL
		p.stateMutex.Unlock()

		p.triggerConfigurationPull()
	} else {
		p.stateMutex.Unlock()
	}
}
