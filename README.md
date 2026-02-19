# YAML Schema Router (yaml-schema-router)

## The Problem

When using the
[`yaml-language-server`](https://github.com/redhat-developer/yaml-language-server)
outside of VSCode (in editors like Helix or Neovim), getting accurate validation
for specific YAML formats is notoriously frustrating.

Most editors rely on simple file extensions or static glob patterns to assign
schemas. If you map a blanket schema to your files, you often get strict
conflicts (e.g., `"Matches multiple schemas when only one must validate"`). If
you don't map a schema, the server just checks for basic YAML syntax and ignores
type-specific validation entirely.

Existing workarounds require either strict, repetitive file-naming conventions
or manually typing modeline comments (`# yaml-language-server: $schema=...`) at
the top of every single file.

## The Solution

`yaml-schema-router` is a lightweight, editor-agnostic standard I/O (stdio)
proxy that sits between your text editor and the `yaml-language-server`. It
intercepts LSP (Language Server Protocol) traffic to dynamically route the
exact, isolated JSON schema to the language server based on the **file's content
and directory context**.

With this wrapper, you get zero-configuration, highly accurate autocomplete,
hover, and validation for supported formats in any editor, without relying on
ambiguous file extensions.

## How It Works

The wrapper acts as a transparent "Man-in-the-Middle" for the JSON-RPC
communication between the editor and the LSP (`yaml-language-server`):

1. **Traffic Interception:** The proxy listens to standard input from the
   editor. For most LSP methods, it blindly passes the JSON payload directly to
   the `yaml-language-server` with zero latency.
2. **Context Sniffing:** When the proxy detects a `textDocument/didOpen` or
   `textDocument/didChange` event, it intercepts the payload to analyze the
   file's raw text and its file path.
3. **Detector Chain:** It runs the file through a chain of "detectors" to
   identify the file type using the most reliable method for that format (e.g.,
   inspecting `apiVersion`/`kind` for K8s or directory paths for GitHub
   Workflows).
4. **Registry Lookup & Caching:** Once identified, the router checks its
   internal **Schema Registry**.
   - If the schema is already cached locally, it is served immediately
     (file-system speed).
   - If not, it is downloaded once and stored in the cache.
5. **Dynamic Schema Injection:** The proxy updates the language server's
   configuration for that specific file URI, pointing it to the locally cached
   schema.

## Key Features

- **True Editor Agnosticism:** Works with Helix, Neovim, Emacs, or any editor
  that supports standard LSP configuration.
- **Smart Detection:** Identifies files based on their actual content or
  directory location rather than just their extension.
- **Local Schema Registry:** Built-in registry automatically downloads and
  caches schemas to your disk.
  - **Offline Development:** Once a schema is fetched, it is available forever,
    allowing you to work without an internet connection.
  - **High Performance:** Caching eliminates network latency on subsequent file
    opens, significantly speeding up schema injection.
- **No Modeline Clutter:** Keeps your files clean by eliminating the need for
  `# yaml-language-server: $schema=...` comments.

## Usage

Configure your editor to use `yaml-schema-router` as the language server
executable instead of `yaml-language-server`.

### Default Behavior

By default, the proxy automatically sets the following `yaml` configurations to
`true` if they are not explicitly defined in your editor's configuration:

- `hover`
- `completion`
- `validation`

> [!IMPORTANT]
> If you wish to disable any of these features, you must explicitly set them to
> `false` in your editor's LSP settings. Whatever reason you might have for
> doing so.

### Command Line Flags

The router accepts the following flags to customize its behavior:

| Flag         | Description                                                                                                                    | Default                                                                                                                                                                  |
| :----------- | :----------------------------------------------------------------------------------------------------------------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--lsp-path` | Path to the underlying `yaml-language-server` executable. Use this if the server is not in your systems PATH.                  | `yaml-language-server`                                                                                                                                                   |
| `--log-file` | Path to a file where logs should be written. **Note:** Since the router communicates via Stdio, logs cannot be sent to stdout. | `~/.cache/yaml-schema-router/router.log` (Linux)<br>`~/Library/Caches/yaml-schema-router/router.log` (macOS)<br>`%LocalAppData%\yaml-schema-router\router.log` (Windows) |

### Example Editor Configuration (Helix)

In your `languages.toml`:

```toml
[[language]]
name = "yaml"
language-servers = [ "yaml-schema-router" ]

[language-server.yaml-schema-router]
command = "yaml-schema-router"
args = [
  "--log-file", "/tmp/yaml-router.log",
  "--lsp-path", "/usr/bin/yaml-language-server"
]

# Explicitly override the proxy defaults
[language-server.yaml-schema-router.config.yaml]
hover = false
completion = false
validation = false
```

## Compatibility

This tool is designed to wrap the
[**Red Hat YAML Language Server**](https://github.com/redhat-developer/yaml-language-server).

- **Recommended Version**: `1.10.0` or higher.
- **Minimum Version**: `1.0.0` (Recommended for full LSP stability).
- **Tested With**: `1.19.0+`.

While the router works with most versions that support the `yaml.schemas`
configuration setting, using a version **1.0+** ensures the best compatibility
with modern LSP clients and disables legacy behaviors (like implicit Kubernetes
schema associations) that this router is designed to replace.

## Supported Detectors

### Kubernetes & CRDs

- **Standard Resources:** Automatically maps standard K8s objects (Deployments,
  Services, etc.) to the correct schema for your version.
- **CRD Support:** Automatically maps custom `apiVersion` and `kind` definitions
  to an internal Custom Resource Definition (CRD) schema registry.

## Roadmap

We are actively working on adding detectors for the following formats:

- [ ] **Config File Support** (Replace flags with a persistent configuration
      file)
- [ ] **GitHub Workflows** (detecting files inside `.github/workflows`)
- [ ] **GitLab CI/CD** (detecting `gitlab-ci.yml` syntax)
- [ ] **Docker Compose** (detecting `services`, `version`, `networks` roots)
- [ ] **Ansible Playbooks/Roles**
- [ ] **Helm Charts** (`Chart.yaml` and `values.yaml`)
- [ ] **OpenAPI/Swagger**
- [ ] **AsyncAPI**
