package lspproxy

import "encoding/json"

// --- Inbound from Editor ---

// BaseRPC represents a generic JSON-RPC 2.0 message, used for peeking at
// incoming/outgoing LSP messages and selectively modifying them.
type BaseRPC struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

// DidOpenNotification represents an incoming textDocument/didOpen LSP message.
type DidOpenNotification struct {
	Method string        `json:"method"`
	Params DidOpenParams `json:"params"`
}

// DidOpenParams holds the parameters for a textDocument/didOpen notification.
type DidOpenParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// TextDocumentItem contains the URI and full text content of an opened document.
type TextDocumentItem struct {
	URI  string `json:"uri"`
	Text string `json:"text"`
}

// DidChangeNotification represents an incoming textDocument/didChange LSP message.
type DidChangeNotification struct {
	Method string          `json:"method"`
	Params DidChangeParams `json:"params"`
}

// DidChangeParams holds the parameters for a textDocument/didChange notification.
type DidChangeParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// VersionedTextDocumentIdentifier identifies a specific document by its URI.
type VersionedTextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// TextDocumentContentChangeEvent contains the modified text of a document.
type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

// --- Outbound to Server (Schema Injection) ---

// DidChangeConfigurationNotification represents an outgoing workspace/didChangeConfiguration message.
type DidChangeConfigurationNotification struct {
	JSONRPC string                       `json:"jsonrpc"`
	Method  string                       `json:"method"`
	Params  DidChangeConfigurationParams `json:"params"`
}

// DidChangeConfigurationParams holds the configuration update payload for the server.
type DidChangeConfigurationParams struct {
	Settings Settings `json:"settings"`
}

// Settings wraps the general configuration settings sent to the language server.
type Settings struct {
	YAML YAMLSound `json:"yaml"`
}

// YAMLSound contains YAML-specific settings, specifically the dynamic schema mappings.
type YAMLSound struct {
	Schemas map[string][]string `json:"schemas"`
}
