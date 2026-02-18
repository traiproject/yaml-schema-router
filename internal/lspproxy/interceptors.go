package lspproxy

import (
	"encoding/json"
	"log"
	"strings"
)

// interceptWorkspaceConfiguration dynamically injects schema configurations
// into the editor's response to the language server.
func (p *Proxy) interceptWorkspaceConfiguration(msg *BaseRPC, payload []byte) []byte {
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
