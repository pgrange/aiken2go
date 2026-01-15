# aiken2go

Generate Go code from Aiken's CIP-0057 Plutus Blueprint.

## Overview

`aiken2go` is a code generator that reads Aiken's `plutus.json` blueprint files and generates Go structs and validator constructors for use in Cardano applications.

## Requirements

- Go 1.21 or later

## Installation

```bash
go install github.com/pgrange/aiken_to_go/cmd/aiken2go@latest
```

Or build from source:

```bash
go build -o aiken2go ./cmd/aiken2go
```

## Usage

Basic usage:

```bash
aiken2go -o contracts.go plutus.json
```

With traced blueprint (for debug scripts):

```bash
aiken2go -o contracts.go -t plutus-trace.json plutus.json
```

With custom package name:

```bash
aiken2go -o contracts.go -p mypackage plutus.json
```

### Options

| Flag | Description |
|------|-------------|
| `-o`, `-outfile` | Output file path (required) |
| `-t`, `-traced-blueprint` | Traced blueprint file for debug scripts (optional) |
| `-p`, `-package` | Go package name (default: `contracts`) |

## Testing

Run all tests:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test ./... -v
```

Run specific test:

```bash
go test ./pkg/blueprint -run TestLoadBlueprint_Simple -v
```

## Test Data

The `testdata/` directory contains sample Plutus blueprints for testing:

```
testdata/
├── simple/
│   ├── plutus.json        # Basic validators with parameters
│   └── plutus-trace.json  # Traced version for debug builds
├── complex/
│   └── plutus.json        # Enum types, nested modules, custom types
└── tuple/
    └── plutus.json        # Tuple types (items as array)
```

### Test Coverage

| Test File | Description |
|-----------|-------------|
| `simple/` | Basic validators, parameters, List and primitive types |
| `complex/` | Enum types (anyOf), single-constructor structs, nested paths |
| `tuple/` | Tuple types where `items` is an array of schemas |

## Project Structure

```
.
├── cmd/
│   └── aiken2go/
│       └── main.go          # CLI entry point
├── pkg/
│   └── blueprint/
│       ├── blueprint.go     # Blueprint loading and types
│       ├── schema.go        # Schema types and helpers
│       ├── generator.go     # Go code generation
│       └── generator_test.go
├── testdata/                # Test blueprints
└── README.md
```

## Generated Code

The generator produces:

- **Type definitions** for custom types in the blueprint
- **Enum interfaces** for types with multiple constructors (anyOf)
- **Struct types** for single-constructor types
- **Validator structs** with `Script` (compiled CBOR hex) and `ScriptHash`
- **Constructor functions** for each validator

Example output:

```go
package contracts

import "math/big"

// MultisigScript enum type
type MultisigScript interface {
    isMultisigScript()
}

type MultisigScriptSignature struct {
    Value string `cbor:"0,keyasint"`
}
func (MultisigScriptSignature) isMultisigScript() {}

// Validator
type MyValidatorSpend struct {
    Script     string
    ScriptHash string
}

func NewMyValidatorSpend(param *big.Int, trace bool) *MyValidatorSpend {
    // ...
}
```

## License

Apache-2.0
