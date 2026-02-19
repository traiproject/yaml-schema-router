package lspproxy

import (
	"encoding/json"
	"log"
	"strings"

	"go.trai.ch/yaml-schema-router/internal/config"
)

// interceptWorkspaceConfiguration dynamically injects schema configurations
// into the editor's response to the language server.
func (p *Proxy) interceptWorkspaceConfiguration(msg *BaseRPC, payload []byte) []byte {
	var result []any
	if err := json.Unmarshal(msg.Result, &result); err != nil || len(result) == 0 {
		return payload
	}

	// Ensure we have a map to work with, even if the editor returned null.
	if result[0] == nil {
		result[0] = make(map[string]any)
	}

	// The first item in the array is the "yaml" section requested by the server
	yamlConfig, ok := result[0].(map[string]any)
	if !ok {
		return payload
	}

	modified := false

	featureDefaults := map[string]bool{
		"hover":      config.DefaultHover,
		"completion": config.DefaultCompletion,
		"validation": config.DefaultValidation,
	}

	// Inject defaults only if the key does not already exist in the user's config.
	for key, defaultValue := range featureDefaults {
		if _, exists := yamlConfig[key]; !exists {
			yamlConfig[key] = defaultValue
			modified = true
		}
	}

	p.stateMutex.RLock()
	groupedSchemas := make(map[string][]string)
	for uri, schemaURL := range p.schemaState {
		groupedSchemas[schemaURL] = append(groupedSchemas[schemaURL], uri)
	}
	p.stateMutex.RUnlock()

	// If no schemas are detected yet, return unmodified payload
	if len(groupedSchemas) == 0 {
		log.Printf("[%s] Intercepted workspace/configuration, but no schemas detected to inject.", componentName)
		modified = true
	}

	if !modified {
		return payload
	}

	log.Printf("[%s] Injecting schemas into workspace/configuration: %v", componentName, groupedSchemas)

	// Inject our schemas into Helix's response
	yamlConfig["schemas"] = groupedSchemas
	result[0] = yamlConfig

	newResult, err := json.Marshal(result)
	if err != nil {
		log.Printf("[%s] Error re-marshaling configuration result: %v", componentName, err)
		return payload
	}
	msg.Result = newResult

	modifiedPayload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[%s] Error re-marshaling configuration payload: %v", componentName, err)
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
				log.Printf("[%s] Successfully intercepted 'initialize' and forced textDocumentSync to Full (1)", componentName)
				return newPayload
			}
			log.Printf("[%s] Warning: failed to re-marshal modified capabilities: %v", componentName, resultErr)
		}
	}

	return payload
}
