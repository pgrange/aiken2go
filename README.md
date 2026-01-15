# aiken2go

Generate Go types from Aiken's CIP-0057 Plutus Blueprint for building Cardano transactions.

## Overview

`aiken2go` reads Aiken's `plutus.json` blueprint files and generates Go types that can be serialized to/from CBOR Plutus Data format. This allows you to construct datums and redeemers in Go for Cardano transactions.

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

```bash
aiken2go -o types.go plutus.json
```

With custom package name:

```bash
aiken2go -o types.go -p mypackage plutus.json
```

### Options

| Flag | Description |
|------|-------------|
| `-o`, `-outfile` | Output file path (required) |
| `-p`, `-package` | Go package name (default: `contracts`) |

## Generated Code

The generator produces:

- **Struct types** for single-constructor types (records)
- **Interface types** for enums (types with multiple constructors)
- **Variant structs** for each enum variant
- **`ToPlutusData()` methods** for serialization
- **`FromPlutusData()` methods** for deserialization
- **`<Type>FromPlutusData()` functions** for enum types

### Example

Given an Aiken type:

```aiken
type PayoutStatus {
  Active
  Paused { reason: ByteArray }
}
```

The generator produces:

```go
// PayoutStatus is an enum type with multiple constructors.
type PayoutStatus interface {
    isPayoutStatus()
    ToPlutusData() (blueprint.PlutusData, error)
}

// PayoutStatusFromPlutusData decodes a PayoutStatus from PlutusData.
func PayoutStatusFromPlutusData(pd blueprint.PlutusData) (PayoutStatus, error) { ... }

type PayoutStatusActive struct{}
func (PayoutStatusActive) isPayoutStatus() {}
func (v PayoutStatusActive) ToPlutusData() (blueprint.PlutusData, error) { ... }
func (v *PayoutStatusActive) FromPlutusData(pd blueprint.PlutusData) error { ... }

type PayoutStatusPaused struct {
    Reason string
}
func (PayoutStatusPaused) isPayoutStatus() {}
func (v PayoutStatusPaused) ToPlutusData() (blueprint.PlutusData, error) { ... }
func (v *PayoutStatusPaused) FromPlutusData(pd blueprint.PlutusData) error { ... }
```

### Using Generated Types

```go
import (
    "github.com/pgrange/aiken_to_go/pkg/blueprint"
    "myproject/contracts" // generated code
)

// Create a datum
datum := contracts.MyDatum{
    Owner: "deadbeef...",
    Amount: big.NewInt(1000000),
}

// Serialize to CBOR for transaction
pd, err := datum.ToPlutusData()
if err != nil {
    return err
}
cborBytes, err := pd.MarshalCBOR()

// Or get hex string
hexString, err := pd.ToHex()

// Deserialize from CBOR
var decoded contracts.MyDatum
var pd blueprint.PlutusData
pd.UnmarshalCBOR(cborBytes)
decoded.FromPlutusData(pd)
```

## PlutusData Format

The CBOR encoding follows the Plutus Data format:

| Plutus Type | CBOR Encoding |
|------------|---------------|
| Integer | CBOR integer/bignum |
| ByteString | CBOR bytes |
| List | CBOR array |
| Map | CBOR map |
| Constructor 0-6 | CBOR tag 121-127 + array |
| Constructor 7+ | CBOR tag 1280+n + array |

## Testing

Run all tests:

```bash
go test ./...
```

Run with verbose output:

```bash
go test ./... -v
```

## Test Data

The `testdata/` directory contains sample Plutus blueprints:

```
testdata/
├── simple/
│   └── plutus.json        # Basic validators and types
├── complex/
│   └── plutus.json        # Enum types, nested modules
└── tuple/
    └── plutus.json        # Tuple types (items as array)
```

## Project Structure

```
.
├── cmd/
│   └── aiken2go/
│       └── main.go              # CLI entry point
├── pkg/
│   └── blueprint/
│       ├── blueprint.go         # Blueprint loading
│       ├── schema.go            # Schema types
│       ├── plutusdata.go        # PlutusData CBOR encoding
│       ├── generator.go         # Go code generation
│       └── *_test.go
├── testdata/                    # Test blueprints
└── README.md
```

## Limitations

- Map types (`Pairs$`) are not fully supported for serialization (TODO comments generated)
- Tuple types are represented as `[]interface{}`

## License

Apache-2.0
