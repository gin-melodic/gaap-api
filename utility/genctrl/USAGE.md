# Protobuf to GoFrame Wrapper Generator

A utility that generates GoFrame-compatible API wrapper files from existing Protobuf service definitions, enabling compatibility with the `gf gen ctrl` command.

## Overview

GoFrame's `gf gen ctrl` command requires API definitions with `g.Meta` annotations. This tool bridges Protobuf-based APIs with GoFrame by generating wrapper structs that embed Protobuf types while adding the necessary GoFrame metadata.

## Usage

### Basic Usage

Run from the project root directory:

```bash
go run utility/genctrl/main.go
```

This uses the default configuration:
- Proto directory: `manifest/protobuf`
- Output directory: `api`

### Custom Paths

```bash
go run utility/genctrl/main.go [proto_dir] [output_dir]
```

Example:
```bash
go run utility/genctrl/main.go manifest/protobuf api
```

## How It Works

### 1. Proto Parsing

The tool scans `.proto` files and extracts:
- Package name (e.g., `account.v1`)
- Service name (e.g., `AccountService`)
- RPC methods with request/response types
- Method comments for API summaries

### 2. HTTP Route Inference

HTTP methods and paths are inferred from RPC method names:

| RPC Method Prefix | HTTP Method | Path Pattern        |
|-------------------|-------------|---------------------|
| `List*`           | GET         | `/{module}`         |
| `Get*`            | GET         | `/{module}/:id`     |
| `Create*`         | POST        | `/{module}`         |
| `Update*`         | PUT         | `/{module}/:id`     |
| `Delete*`         | DELETE      | `/{module}/:id`     |
| Other             | POST        | `/{module}/{method}`|

### 3. Wrapper Generation

For each proto service, generates a Go file with wrapper types:

```go
// Original Protobuf type: ListAccountsReq
// Generated wrapper:
type ListAccountsReq struct {
    g.Meta `path:"/account" method:"GET" tags:"account" summary:"List all accounts"`
    *pb.ListAccountsReq
}

type ListAccountsRes = pb.ListAccountsRes
```

## Output Structure

```
api/
├── account/
│   └── v1/
│       ├── account.pb.go       # Original (protoc generated)
│       ├── account_grpc.pb.go  # Original (protoc generated)
│       └── account.go          # Generated wrapper
├── auth/
│   └── v1/
│       └── auth.go             # Generated wrapper
└── ...
```

## After Generation

Run GoFrame's controller generator:

```bash
gf gen ctrl
```

This will generate controller scaffolding in `internal/controller/`.

## Notes

- The `base.proto` file is skipped as it contains only common types
- Generated files have a `DO NOT EDIT` header
- Existing `.pb.go` files are preserved
- Wrapper types embed the original Protobuf types for full compatibility
