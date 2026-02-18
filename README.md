# YAML LSP Schema Proxy (yaml-schema-router)

## The Problem

When using the `yaml-language-server` outside of VSCode (in editors like Helix
or Neovim), getting accurate Kubernetes manifest validation is a notoriously
frustrating Catch-22.

If you map a blanket Kubernetes schema (like `all.json`) to your YAML files, the
language server fails with a strict `oneOf` conflict
(`"Matches multiple schemas when only one must validate"`). If you don't map a
schema, the server just checks for basic YAML syntax and ignores Kubernetes
specific validation entirely. Existing workarounds require either strict,
repetitive file-naming conventions (e.g., `*namespace.yaml`) or manually typing
modeline comments (`# yaml-language-server: $schema=...`) at the top of every
single manifest.

## The Solution

`yaml-schema-router` is a lightweight, editor-agnostic standard I/O (stdio)
proxy that sits between your text editor and the `yaml-language-server`. It
intercepts LSP (Language Server Protocol) traffic to dynamically route the
exact, isolated Kubernetes JSON schema to the language server based purely on
the file's content.

With this wrapper, you get zero-configuration, highly accurate Kubernetes
autocomplete, hover, and validation in any editor, regardless of what you name
your files.

## How It Works

The wrapper acts as a transparent "Man-in-the-Middle" for the JSON-RPC
communication between the editor (Helix) and the LSP (`yaml-language-server`):

1. **Traffic Interception:** The proxy listens to standard input from the
   editor. For most LSP methods (like `textDocument/hover` or
   `textDocument/completion`), it blindly passes the JSON payload directly to
   the `yaml-language-server` with zero latency.
2. **Content Sniffing:** When the proxy detects a `textDocument/didOpen` or
   `textDocument/didChange` event, it intercepts the payload and reads the raw
   text of the file.
3. **Resource Detection:** It quickly parses the YAML text specifically looking
   for the `apiVersion` and `kind` keys (e.g., `apiVersion: apps/v1`,
   `kind: Deployment`).
4. **Dynamic Schema Injection:** Once the resource type is identified, the proxy
   maps it to a specific, isolated JSON schema URL (e.g.,
   `deployment-apps-v1.json`). It then dynamically updates the language server's
   configuration for that specific file URI in memory, telling the LSP exactly
   which schema to use before passing the file content along.

## Key Features

- **True Editor Agnosticism:** Works with Helix, Neovim, Emacs, or any editor
  that supports standard LSP configuration.
- **No File Naming Rules:** Name your manifests whatever you want
  (`backend.yaml`, `prod-db.yml`). The proxy relies on the file's actual
  content, not its extension.
- **No Modeline Clutter:** Keeps your manifests clean by eliminating the need
  for `# yaml-language-server: $schema=...` comments.
- **CRD Support:** Can be extended to automatically map custom `apiVersion` and
  `kind` definitions to an internal Custom Resource Definition (CRD) schema
  registry.
