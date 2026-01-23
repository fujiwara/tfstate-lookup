# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./tfstate

# Run a specific test
go test -run TestLookup ./tfstate

# Install the CLI
go install github.com/fujiwara/tfstate-lookup/cmd/tfstate-lookup

# Format code (required before commit)
go fmt ./...

# Build with selective backends (exclude unused backends)
go build -tags no_gcs,no_azurerm,no_tfe ./...
```

## Architecture Overview

tfstate-lookup is a Go tool and library for looking up Terraform state file resources. It supports both CLI usage and programmatic usage as a Go package.

### Package Structure

```
cmd/tfstate-lookup/main.go  # CLI entry point
tfstate/
├── lookup.go               # Main TFState, Object types and Read/ReadURL functions
├── funcs.go                # Template FuncMap for tfstate lookups
├── jsonnet.go              # Jsonnet native functions
├── remote.go               # Remote backend dispatch
├── remote_http.go          # HTTP backend (always available)
├── remote_s3.go            # S3 backend (build tag: !no_s3)
├── remote_s3_stub.go       # S3 stub when excluded (build tag: no_s3)
├── remote_gcs.go           # GCS backend (build tag: !no_gcs)
├── remote_gcs_stub.go      # GCS stub when excluded (build tag: no_gcs)
├── remote_azurerm.go       # Azure backend (build tag: !no_azurerm)
├── remote_azurerm_stub.go  # Azure stub when excluded (build tag: no_azurerm)
├── remote_tfe.go           # TFE backend (build tag: !no_tfe)
└── remote_tfe_stub.go      # TFE stub when excluded (build tag: no_tfe)
```

### Core Components

- **TFState**: Main struct with `Lookup()`, `List()`, `Dump()` methods
- **Object**: Wrapper for lookup results with JSON serialization and gojq query support
- **ReadURL()**: Entry point supporting multiple URL schemes

### Selective Backend Build (Build Tags)

For smaller binaries, use build tags to exclude unused backends:

```bash
# Exclude GCS, AzureRM, and TFE backends (keep only S3)
go build -tags no_gcs,no_azurerm,no_tfe ./...

# Exclude all cloud backends
go build -tags no_s3,no_gcs,no_azurerm,no_tfe ./...
```

Available build tags:
- `no_s3` - Exclude AWS S3 backend
- `no_gcs` - Exclude Google Cloud Storage backend
- `no_azurerm` - Exclude Azure Blob Storage backend
- `no_tfe` - Exclude Terraform Cloud/Enterprise backend

### Supported URL Schemes

- `file://path` or plain path: Local file
- `http://` / `https://`: HTTP fetch (always available, stdlib only)
- `s3://bucket/key`: AWS S3
- `gs://bucket/key`: Google Cloud Storage
- `azurerm://resource-group/storage-account/container/blob`: Azure Blob Storage
- `remote://host/org/workspace`: Terraform Cloud/Enterprise

### Key Environment Variables

- `TF_WORKSPACE`: Terraform workspace name
- `TFE_TOKEN`: Terraform Cloud/Enterprise API token
- `AWS_ENDPOINT_URL_S3`: Custom S3 endpoint for S3-compatible storage
- `GOOGLE_ENCRYPTION_KEY`: GCS customer-supplied encryption key
