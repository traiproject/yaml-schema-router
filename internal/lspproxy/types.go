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
