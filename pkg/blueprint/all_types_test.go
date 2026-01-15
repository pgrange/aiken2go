package blueprint

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestAllTypesRoundTrip tests that all Aiken types can be correctly
// marshaled to CBOR PlutusData and unmarshaled back.
// This validates the end-to-end flow with a real Aiken-generated blueprint.
func TestAllTypesRoundTrip(t *testing.T) {
	// Create temp directory for generated code
	tmpDir, err := os.MkdirTemp("", "all_types_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectory for generated types
	typesDir := filepath.Join(tmpDir, "types")
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		t.Fatalf("failed to create types dir: %v", err)
	}

	// Generate Go code from the all_types blueprint
	bp, err := LoadBlueprint("../../testdata/all_types/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, GeneratorOptions{PackageName: "types"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Write the generated types to types/types.go
	typesFile := filepath.Join(typesDir, "types.go")
	if err := os.WriteFile(typesFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write types file: %v", err)
	}

	// Write the test program
	testProgram := `package main

import (
	"fmt"
	"math/big"
	"os"

	"github.com/pgrange/aiken_to_go/pkg/blueprint"
	"testpkg/types"
)

func testRoundTrip[T interface {
	ToPlutusData() (blueprint.PlutusData, error)
}](name string, original T, decode func(blueprint.PlutusData) (T, error)) error {
	// Serialize to PlutusData
	pd, err := original.ToPlutusData()
	if err != nil {
		return fmt.Errorf("%s ToPlutusData: %v", name, err)
	}

	// Serialize to CBOR
	cborBytes, err := pd.MarshalCBOR()
	if err != nil {
		return fmt.Errorf("%s MarshalCBOR: %v", name, err)
	}

	// Deserialize from CBOR
	var decodedPd blueprint.PlutusData
	if err := decodedPd.UnmarshalCBOR(cborBytes); err != nil {
		return fmt.Errorf("%s UnmarshalCBOR: %v", name, err)
	}

	// Deserialize back to type
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

	// Test SimpleString
	simpleString := types.SimpleString{Message: "48656c6c6f"} // "Hello"
	if err := testRoundTrip("SimpleString", simpleString, func(pd blueprint.PlutusData) (types.SimpleString, error) {
		var v types.SimpleString
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test SimpleInt
	simpleInt := types.SimpleInt{Value: big.NewInt(42)}
	if err := testRoundTrip("SimpleInt", simpleInt, func(pd blueprint.PlutusData) (types.SimpleInt, error) {
		var v types.SimpleInt
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test MultipleFields with Bool
	multipleFields := types.MultipleFields{
		Name:   "416c696365", // "Alice"
		Age:    big.NewInt(30),
		Active: true,
	}
	if err := testRoundTrip("MultipleFields", multipleFields, func(pd blueprint.PlutusData) (types.MultipleFields, error) {
		var v types.MultipleFields
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithList (list of ints)
	withList := types.WithList{
		Items: []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)},
	}
	if err := testRoundTrip("WithList", withList, func(pd blueprint.PlutusData) (types.WithList, error) {
		var v types.WithList
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithStringList (list of strings)
	withStringList := types.WithStringList{
		Names: []string{"416c696365", "426f62"}, // "Alice", "Bob"
	}
	if err := testRoundTrip("WithStringList", withStringList, func(pd blueprint.PlutusData) (types.WithStringList, error) {
		var v types.WithStringList
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithOption (with Some value)
	withOptionSome := types.WithOption{
		Label:      "6c6162656c", // "label"
		MaybeValue: types.OptionInt{Value: big.NewInt(100), IsSet: true},
	}
	if err := testRoundTrip("WithOption(Some)", withOptionSome, func(pd blueprint.PlutusData) (types.WithOption, error) {
		var v types.WithOption
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithOption (with None)
	withOptionNone := types.WithOption{
		Label:      "6c6162656c", // "label"
		MaybeValue: types.OptionInt{IsSet: false},
	}
	if err := testRoundTrip("WithOption(None)", withOptionNone, func(pd blueprint.PlutusData) (types.WithOption, error) {
		var v types.WithOption
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test Status enum - Active variant
	statusActive := types.StatusActive{}
	if err := testRoundTrip("StatusActive", statusActive, func(pd blueprint.PlutusData) (types.StatusActive, error) {
		var v types.StatusActive
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test Status enum - Inactive variant
	statusInactive := types.StatusInactive{}
	if err := testRoundTrip("StatusInactive", statusInactive, func(pd blueprint.PlutusData) (types.StatusInactive, error) {
		var v types.StatusInactive
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test Status enum - Pending variant with data
	statusPending := types.StatusPending{Reason: "74657374"} // "test"
	if err := testRoundTrip("StatusPending", statusPending, func(pd blueprint.PlutusData) (types.StatusPending, error) {
		var v types.StatusPending
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test Nested type
	nested := types.Nested{
		Inner: types.SimpleString{Message: "696e6e6572"}, // "inner"
		Count: big.NewInt(5),
	}
	if err := testRoundTrip("Nested", nested, func(pd blueprint.PlutusData) (types.Nested, error) {
		var v types.Nested
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithEnum (struct containing enum)
	withEnum := types.WithEnum{
		Id:     "6d7969640a", // "myid"
		Status: types.StatusActive{},
	}
	if err := testRoundTrip("WithEnum", withEnum, func(pd blueprint.PlutusData) (types.WithEnum, error) {
		var v types.WithEnum
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithRecordList (list of structs)
	withRecordList := types.WithRecordList{
		Entries: []types.SimpleString{
			{Message: "6f6e65"},   // "one"
			{Message: "74776f"},  // "two"
		},
	}
	if err := testRoundTrip("WithRecordList", withRecordList, func(pd blueprint.PlutusData) (types.WithRecordList, error) {
		var v types.WithRecordList
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	// Test WithOptionalNested
	withOptionalNested := types.WithOptionalNested{
		Data: types.OptionSimpleString{
			Value: types.SimpleString{Message: "6f7074"},
			IsSet: true,
		},
	}
	if err := testRoundTrip("WithOptionalNested", withOptionalNested, func(pd blueprint.PlutusData) (types.WithOptionalNested, error) {
		var v types.WithOptionalNested
		return v, v.FromPlutusData(pd)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	if failed {
		os.Exit(1)
	}
	fmt.Println("\n✓ All round-trip tests passed!")
}
`

	// Write test program to main.go
	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("failed to write main file: %v", err)
	}

	// Get absolute path to project root
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to get project root: %v", err)
	}

	// Initialize go module
	goModContent := `module testpkg

go 1.21

require github.com/pgrange/aiken_to_go v0.0.0

replace github.com/pgrange/aiken_to_go => ` + projectRoot + `
`
	goModFile := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, output)
	}

	// Run the test program
	cmd = exec.Command("go", "run", "main.go")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("test program failed: %v\n%s", err, output)
	}

	t.Logf("Test output:\n%s", output)
}
