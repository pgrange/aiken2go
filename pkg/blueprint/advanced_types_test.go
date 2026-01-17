package blueprint

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestAdvancedTypes tests the code generation for advanced type patterns:
// - Primitive wrappers (types that wrap bytes/integer like PolicyId, AssetName)
// - Bool refs (fields that reference Bool type)
// - Data type (raw PlutusData)
// - Tuple types (lists with multiple items)
// - Enums in Option types
// - Enums in wrapper types (single-field variants)
func TestAdvancedTypes(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/advanced_types/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "advanced"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Test 1: Primitive wrapper types should be generated as string (for bytes) or *big.Int (for integer)
	t.Run("PrimitiveWrappers", func(t *testing.T) {
		// Token struct should have PolicyId as string, not as a custom type
		if !strings.Contains(code, "PolicyId string") {
			t.Error("Expected PolicyId field to be 'string' type")
		}
		if !strings.Contains(code, "AssetName string") {
			t.Error("Expected AssetName field to be 'string' type")
		}
		if !strings.Contains(code, "Amount *big.Int") {
			t.Error("Expected Amount field to be '*big.Int' type")
		}
	})

	// Test 2: Bool refs should generate inline bool handling
	t.Run("BoolRefs", func(t *testing.T) {
		if !strings.Contains(code, "Active bool") {
			t.Error("Expected Active field to be 'bool' type")
		}
		// Check for proper bool serialization (constructor 0 or 1)
		if !strings.Contains(code, "NewConstrPlutusData(1)") && !strings.Contains(code, "NewConstrPlutusData(0)") {
			t.Error("Expected bool serialization with constructor 0/1")
		}
	})

	// Test 3: Data type should be interface{}
	t.Run("DataType", func(t *testing.T) {
		if !strings.Contains(code, "Data interface{}") {
			t.Error("Expected Data field to be 'interface{}' type")
		}
		// Check for proper Data handling (type assertion to PlutusData)
		if !strings.Contains(code, ".(PlutusData)") {
			t.Error("Expected type assertion to PlutusData for Data type")
		}
	})

	// Test 4: Tuple types should be generated as structs with Field0, Field1, etc.
	t.Run("TupleTypes", func(t *testing.T) {
		// Asset tuple type
		if !strings.Contains(code, "type Asset struct") {
			t.Error("Expected Asset tuple type to be generated as struct")
		}
		// Check for tuple fields
		if !strings.Contains(code, "Field0 string") {
			t.Error("Expected Field0 in tuple struct")
		}
		if !strings.Contains(code, "Field1 string") {
			t.Error("Expected Field1 in tuple struct")
		}
		// Tuple should use list serialization
		if !strings.Contains(code, "NewListPlutusData(items...)") {
			t.Error("Expected tuple to use list serialization")
		}
	})

	// Test 5: Option with enum inner type should use factory function
	t.Run("OptionWithEnum", func(t *testing.T) {
		// Check for factory function usage
		if !strings.Contains(code, "CredentialFromPlutusData(") {
			t.Error("Expected CredentialFromPlutusData factory function to be used")
		}
	})

	// Test 6: Enum wrapper types (single-field variants with primitive wrapper)
	t.Run("EnumWrapperWithPrimitive", func(t *testing.T) {
		// CredentialVerificationKey should have Value string (KeyHash is bytes)
		if !strings.Contains(code, "type CredentialVerificationKey struct") {
			t.Error("Expected CredentialVerificationKey struct to be generated")
		}
	})

	// Test 7: Lists of primitive wrappers
	t.Run("ListOfPrimitiveWrappers", func(t *testing.T) {
		if !strings.Contains(code, "Policies []string") {
			t.Error("Expected Policies to be []string (list of PolicyId which is bytes)")
		}
	})

	// Test 8: Named list types (like SignatureList)
	t.Run("NamedListType", func(t *testing.T) {
		if !strings.Contains(code, "type SignatureList []Credential") {
			t.Error("Expected SignatureList to be generated as type alias for []Credential")
		}
		// Check for ToPlutusData method
		if !strings.Contains(code, "func (v SignatureList) ToPlutusData()") {
			t.Error("Expected SignatureList to have ToPlutusData method")
		}
		// Check for FromPlutusData method
		if !strings.Contains(code, "func (v *SignatureList) FromPlutusData(") {
			t.Error("Expected SignatureList to have FromPlutusData method")
		}
	})
}

// TestAdvancedTypesCompiles verifies that the generated code for advanced types compiles
func TestAdvancedTypesCompiles(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go compiler not found, skipping compilation test")
	}

	tmpDir, err := os.MkdirTemp("", "advanced_types_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	bp, err := LoadBlueprint("../../testdata/advanced_types/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "advanced"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	outFile := filepath.Join(tmpDir, "advanced.go")
	if err := os.WriteFile(outFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write generated code: %v", err)
	}

	goMod := `module testmod

go 1.21

require github.com/fxamacker/cbor/v2 v2.8.0

require github.com/x448/float16 v0.8.4 // indirect
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\nOutput: %s", err, output)
	}

	cmd := exec.Command("go", "build", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("generated code failed to compile: %v\nOutput: %s\n\nGenerated code:\n%s", err, output, code)
	}
}

// TestAdvancedTypesRoundTrip tests serialization and deserialization of advanced types
func TestAdvancedTypesRoundTrip(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "advanced_types_roundtrip")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	typesDir := filepath.Join(tmpDir, "types")
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		t.Fatalf("failed to create types dir: %v", err)
	}

	bp, err := LoadBlueprint("../../testdata/advanced_types/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "types"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	typesFile := filepath.Join(typesDir, "types.go")
	if err := os.WriteFile(typesFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write types file: %v", err)
	}

	testProgram := `package main

import (
	"fmt"
	"math/big"
	"os"

	"testpkg/types"
)

func testRoundTrip[T interface {
	ToPlutusData() (types.PlutusData, error)
}](name string, original T, decode func(types.PlutusData) (T, error)) error {
	pd, err := original.ToPlutusData()
	if err != nil {
		return fmt.Errorf("%s ToPlutusData: %v", name, err)
	}

	cborBytes, err := pd.MarshalCBOR()
	if err != nil {
		return fmt.Errorf("%s MarshalCBOR: %v", name, err)
	}

	var decodedPd types.PlutusData
	if err := decodedPd.UnmarshalCBOR(cborBytes); err != nil {
		return fmt.Errorf("%s UnmarshalCBOR: %v", name, err)
	}

	_, err = decode(decodedPd)
	if err != nil {
		return fmt.Errorf("%s FromPlutusData: %v", name, err)
	}

	hex, _ := pd.ToHex()
	fmt.Printf("✓ %s: %s\n", name, hex)
	return nil
}

func main() {
	var failed bool

	// Test Token with primitive wrappers (PolicyId, AssetName as bytes, Amount as int)
	token := types.Token{
		PolicyId:  "abcd1234",
		AssetName: "546f6b656e", // "Token"
		Amount:    big.NewInt(1000),
	}
	if err := testRoundTrip("Token", token, func(pd types.PlutusData) (types.Token, error) {
		var v types.Token
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithBoolRef (Bool as ref)
	withBool := types.WithBoolRef{
		Name:   "74657374", // "test"
		Active: true,
	}
	if err := testRoundTrip("WithBoolRef", withBool, func(pd types.PlutusData) (types.WithBoolRef, error) {
		var v types.WithBoolRef
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test Asset tuple type (values must be valid hex strings)
	asset := types.Asset{
		Field0: "abcdef123456", // policy id as hex
		Field1: "546f6b656e",   // "Token" as hex
	}
	if err := testRoundTrip("Asset", asset, func(pd types.PlutusData) (types.Asset, error) {
		var v types.Asset
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test TupleIntBytearray
	tuple := types.TupleIntBytearray{
		Field0: big.NewInt(42),
		Field1: "48656c6c6f", // "Hello"
	}
	if err := testRoundTrip("TupleIntBytearray", tuple, func(pd types.PlutusData) (types.TupleIntBytearray, error) {
		var v types.TupleIntBytearray
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test Address with Option<Credential> (hex values)
	addr := types.Address{
		Payment: types.CredentialVerificationKey{Value: "aabbccdd1122"},
		Stake:   types.OptionCredential{IsSet: false},
	}
	if err := testRoundTrip("Address", addr, func(pd types.PlutusData) (types.Address, error) {
		var v types.Address
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test Address with Some(Credential) (hex values)
	addrWithStake := types.Address{
		Payment: types.CredentialScript{Value: "112233445566"},
		Stake:   types.OptionCredential{Value: types.CredentialVerificationKey{Value: "778899aabbcc"}, IsSet: true},
	}
	if err := testRoundTrip("Address(with stake)", addrWithStake, func(pd types.PlutusData) (types.Address, error) {
		var v types.Address
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithData (Data type as interface{})
	withData := types.WithData{
		Label: "6c6162656c", // "label"
		Data:  types.NewConstrPlutusData(0, types.NewIntPlutusData(big.NewInt(123))),
	}
	if err := testRoundTrip("WithData", withData, func(pd types.PlutusData) (types.WithData, error) {
		var v types.WithData
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test DatumInlineDatum (enum variant with Data)
	inlineDatum := types.DatumInlineDatum{
		Value: types.NewBytesPlutusData([]byte{1, 2, 3}),
	}
	if err := testRoundTrip("DatumInlineDatum", inlineDatum, func(pd types.PlutusData) (types.DatumInlineDatum, error) {
		var v types.DatumInlineDatum
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithPolicyList (list of primitive wrappers - hex values)
	withPolicyList := types.WithPolicyList{
		Policies: []string{"aabb11", "ccdd22", "eeff33"},
	}
	if err := testRoundTrip("WithPolicyList", withPolicyList, func(pd types.PlutusData) (types.WithPolicyList, error) {
		var v types.WithPolicyList
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	if failed {
		os.Exit(1)
	}
	fmt.Println("\n✓ All advanced types round-trip tests passed!")
}
`

	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("failed to write main file: %v", err)
	}

	goModContent := `module testpkg

go 1.21

require github.com/fxamacker/cbor/v2 v2.8.0

require github.com/x448/float16 v0.8.4 // indirect
`
	goModFile := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, output)
	}

	cmd = exec.Command("go", "run", "main.go")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("test program failed: %v\n%s", err, output)
	}

	t.Logf("Test output:\n%s", output)
}
