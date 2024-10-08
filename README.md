# Reapo

Reapo is an I/O server that retrieves context for a local codebase (including documentation).

## Features

- Communicates via stdin/stdout using JSON-RPC, similar to LSP
- Uses Tree-sitter to parse the codebase for public definitions
- Utilizes LlamaIndex for indexing and retrieving documentation

## Installation

Requires Python 3.10 or higher.

Install the dependencies:

```bash
make install
```

## Useful commands for development

tree -a -I .git -I .venv -I docs
