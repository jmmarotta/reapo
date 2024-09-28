# Reapo Design Document

consider: use avante.nvim repo_map parser to parse the codebase.

Reapo is an attempt at creating an LSP-like protocol for providing
 context for a codebase.

## System Architecture

- communicate like an LSP via stdin/stdout with JSON-RPC
- use tree-sitter to parse the codebase
- llamaIndex

We have the folowing:

- ContextServer

We have the following contexts:

- one for public symbols
- one for documentation

## Protocols

### Servers (Workspaces)

#### initialize

Initializes the server for the current workspace.
If the server is already initialized for the workspace
 it will return the capabilities of that workspace.

**input:**

```json
{
    "jsonrpc": "2.0"
    "method": "initialize",
    "params": {
        "workspaceFolders": [
            {
                "uri": "string",
                "extensions": ["string"]
            }
        ],
    },
    "id": 1
}
```

**output:**

```json
{
    "jsonrpc": "2.0",
    "result": {
        "capabilities": {
            "workspaceFolders": [
                {
                    "uri": "string",
                    "extensions": ["string"]
                }
            ],
            "context": ["repo", "documentation"]
        }
    },
    id: 1
}
```

#### initialized

**input:**

```json
{
    "jsonrpc": "2.0"
    "method": "initialized",
    "params": {
        "workspaceFolders": [
            {
                "uri": "string"
            }
        ],
    }
}
```

**output:**
no output

#### shutdown

**input:**

```json
{
    "jsonrpc": "2.0"
    "method": "shutdown",
    "params": {
        "workspaceFolders": [
            {
                "uri": "string"
            }
        ],
    },
    "id": 1
}
```

**output:**

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": null
}
```

#### exit

**input:**

```json
{
    "jsonrpc": "2.0"
    "method": "exit",
}
```

**output:**
no output

----

### Context Indexing

#### indexGitRepo (probably will be done on initialize)

Starts indexing a git repository.
Uses tree-sitter to parse the codebase for public definitions.

**input:**

```json
{
    "jsonrpc": "2.0"
    "method": "indexGitRepo",
    "params": {
        "directoryPath": "string",
    },
    "id": 1
}
```

**output:**

```json
{
    "jsonrpc": "2.0",
    "result": {
        "indexed": "boolean",
    },
    "id": 1
}
```

#### addFileDocumentationToRepo

Uses LlamaIndex to add documentation for a file to the index.

**input:**

```json
{
    "jsonrpc": "2.0"
    "method": "addFileDocumentationToRepo",
    "params": {
        "filePath": "string",
    },
    "id": 1
}

```

**output:**

```json
{
    "jsonrpc": "2.0",
    "result": {
        "indexed": "boolean",
    },
    "id": 1
}
```

#### addURLDocumentationToRepo

Uses LlamaIndex to add documentation for a URL to the index.

**input:**

```json
{
    "jsonrpc": "2.0"
    "method": "addURLDocumentationToRepo",
    "params": {
        "url": "string",
    },
    "id": 1
}
```

**output:**

```json
{
    "jsonrpc": "2.0",
    "result": {
        "indexed": "boolean",
    },
    "id": 1
}
```

#### addSiteDocumentationToRepo

Uses LlamaIndex to add documentation for a site to the index.

**input:**

```json
{
    "jsonrpc": "2.0"
    "method": "addSiteDocumentationToRepo",
    "params": {
        "url": "string",
    },
    "id": 1
}
```

**output:**

```json
{
    "jsonrpc": "2.0",
    "result": {
        "indexed": "boolean",
    },
    "id": 1
}
```


----

### Context Retrieval

#### retrieveRepoContext

Retrieves context for the current file's repository.

**input:**

```json
{
    "jsonrpc": "2.0"
    "method": "retrieveRepoContext",
    "params": {
        "maxTokens": "number",
        "filePath": "string",
        "cursorPosition": {
            "line": "number",
            "column": "number"
        },
        "prompt": "string"
    },
    "id": 1
}
```

**output:**

```json
{
    "jsonrpc": "2.0",
    "result": {
        "context": "string",
    },
    "id": 1
}
```

#### retrieveDocumentationContext

Retrieves context for the documentation of the current file's repository.

**input:**

```json
{
    "jsonrpc": "2.0"
    "method": "retrieveDocumentationContext",
    "params": {
        "maxTokens": "number",
        "filePath": "string",
        "cursorPosition": {
            "line": "number",
            "column": "number"
        },
        "prompt": "string"
    },
    "id": 1
}
```

**output:**

```json
{
    "jsonrpc": "2.0",
    "result": {
        "context": "string",
    },
    "id": 1
}
```

#### retrieveContext

**input:**

```json
{
    "jsonrpc": "2.0"
    "method": "retrieveContext",
    "params": {
        "maxTokens": "number",
        "filePath": "string",
        "cursorPosition": {
            "line": "number",
            "column": "number"
        },
        "prompt": "string"
    },
    "id": 1
}
```

**output:**

```json
{
    "jsonrpc": "2.0",
    "result": {
        "context": "string",
    },
    "id": 1
}
```
