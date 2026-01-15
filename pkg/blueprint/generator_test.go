package blueprint

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBlueprint_Simple(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/simple/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	// Check preamble
	if bp.Preamble.Title != "blueprint/test" {
		t.Errorf("expected title 'blueprint/test', got %q", bp.Preamble.Title)
	}
	if bp.Preamble.PlutusVersion != "v3" {
		t.Errorf("expected plutusVersion 'v3', got %q", bp.Preamble.PlutusVersion)
	}

	// Check validators count
	if len(bp.Validators) != 6 {
		t.Errorf("expected 6 validators, got %d", len(bp.Validators))
	}

	// Check definitions count
	if len(bp.Definitions) != 4 {
		t.Errorf("expected 4 definitions, got %d", len(bp.Definitions))
	}
}

func TestLoadBlueprint_Complex(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/complex/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	// Check preamble
	if bp.Preamble.Title != "treasury/funds" {
		t.Errorf("expected title 'treasury/funds', got %q", bp.Preamble.Title)
	}

	// Check validators count
	if len(bp.Validators) != 3 {
		t.Errorf("expected 3 validators, got %d", len(bp.Validators))
	}

	// Check we have enum types
	multisig, ok := bp.Definitions["multisig/MultisigScript"]
	if !ok {
		t.Error("expected to find MultisigScript definition")
	}
	if multisig == nil || !multisig.IsEnum() {
		t.Error("MultisigScript should be an enum type")
	}
	if multisig != nil && len(multisig.AnyOf) != 7 {
		t.Errorf("MultisigScript should have 7 variants, got %d", len(multisig.AnyOf))
	}
}

func TestSchema_RefName(t *testing.T) {
	tests := []struct {
		ref      string
		expected string
	}{
		{"#/definitions/Int", "Int"},
		{"#/definitions/ByteArray", "ByteArray"},
		{"#/definitions/types~1Payout", "types/Payout"},
		{"#/definitions/cardano~1assets~1PolicyId", "cardano/assets/PolicyId"},
	}

	for _, tc := range tests {
		s := &Schema{Ref: tc.ref}
		got := s.RefName()
		if got != tc.expected {
			t.Errorf("RefName(%q) = %q, want %q", tc.ref, got, tc.expected)
		}
	}
}

func TestSchema_TypeDetection(t *testing.T) {
	// Test integer
	intSchema := &Schema{DataType: "integer"}
	if !intSchema.IsInteger() {
		t.Error("should detect integer type")
	}

	// Test bytes
	bytesSchema := &Schema{DataType: "bytes"}
	if !bytesSchema.IsBytes() {
		t.Error("should detect bytes type")
	}

	// Test list
	listSchema := &Schema{DataType: "list", Items: &Schema{DataType: "integer"}}
	if !listSchema.IsList() {
		t.Error("should detect list type")
	}

	// Test map
	mapSchema := &Schema{DataType: "map"}
	if !mapSchema.IsMap() {
		t.Error("should detect map type")
	}

	// Test constructor
	idx := 0
	constructorSchema := &Schema{DataType: "constructor", Index: &idx}
	if !constructorSchema.IsConstructor() {
		t.Error("should detect constructor type")
	}

	// Test enum
	enumSchema := &Schema{
		AnyOf: []Schema{
			{Title: "VariantA", DataType: "constructor", Index: &idx},
			{Title: "VariantB", DataType: "constructor"},
		},
	}
	if !enumSchema.IsEnum() {
		t.Error("should detect enum type")
	}

	// Test Unit/Void
	unitSchema := &Schema{
		AnyOf: []Schema{
			{DataType: "constructor", Index: &idx, Fields: []Schema{}},
		},
	}
	if !unitSchema.IsUnit() {
		t.Error("should detect unit type")
	}

	// Test Boolean
	idx1 := 1
	boolSchema := &Schema{
		AnyOf: []Schema{
			{Title: "False", DataType: "constructor", Index: &idx},
			{Title: "True", DataType: "constructor", Index: &idx1},
		},
	}
	if !boolSchema.IsBoolean() {
		t.Error("should detect boolean type")
	}

	// Test Option
	optionSchema := &Schema{
		AnyOf: []Schema{
			{Title: "Some", DataType: "constructor", Index: &idx, Fields: []Schema{{DataType: "integer"}}},
			{Title: "None", DataType: "constructor"},
		},
	}
	if !optionSchema.IsOption() {
		t.Error("should detect option type")
	}
}

func TestGenerateSimple(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/simple/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, nil, GeneratorOptions{PackageName: "contracts"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Check that the code contains expected elements
	checks := []string{
		"package contracts",
		"import (",
		`"math/big"`,
		"type AlwaysTrueScriptSpend struct",
		"type AlwaysTrueScriptElse struct",
		"type AlwaysTrueScriptNoParamsSpend struct",
		"type NestedSometimesTrueScriptSpend struct",
		"func NewAlwaysTrueScriptSpend(",
		"func NewAlwaysTrueScriptNoParamsSpend(",
		"Script     string",
		"ScriptHash string",
	}

	for _, check := range checks {
		if !strings.Contains(code, check) {
			t.Errorf("generated code missing expected element: %q", check)
		}
	}
}

func TestGenerateComplex(t *testing.T) {
	bp, err := LoadBlueprint("../../testdata/complex/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, nil, GeneratorOptions{PackageName: "treasury"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Check for enum types (interface + variants)
	checks := []string{
		"package treasury",
		"type MultisigScript interface",
		"isMultisigScript()",
		"type MultisigScriptSignature struct",
		"type MultisigScriptAllOf struct",
		"type MultisigScriptAnyOf struct",
		"type MultisigScriptAtLeast struct",
		"type MultisigScriptBefore struct",
		"type MultisigScriptAfter struct",
		"type MultisigScriptScript struct",
		"func (MultisigScriptSignature) isMultisigScript()",
		"type PayoutStatus interface",
		"type PayoutStatusActive struct",
		"type PayoutStatusPaused struct",
		"type TreasurySpendRedeemer interface",
		"type TreasurySpendRedeemerReorganize struct",
		"type TreasurySpendRedeemerFund struct",
		"type VendorSpendRedeemer interface",
		// Single constructor types should be structs
		"type Payout struct",
		"type TreasuryConfiguration struct",
		"type TreasuryPermissions struct",
		// Validators
		"type TreasuryTreasurySpend struct",
		"type VendorVendorSpend struct",
		"func NewTreasuryTreasurySpend(",
	}

	for _, check := range checks {
		if !strings.Contains(code, check) {
			t.Errorf("generated code missing expected element: %q", check)
		}
	}
}

func TestGenerateWithTrace(t *testing.T) {
	bp, traceBp, err := LoadBlueprintWithTrace(
		"../../testdata/simple/plutus.json",
		"../../testdata/simple/plutus-trace.json",
	)
	if err != nil {
		t.Fatalf("failed to load blueprints: %v", err)
	}

	gen := NewGenerator(bp, traceBp, GeneratorOptions{PackageName: "contracts", WithTrace: true})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Check for trace parameter handling
	checks := []string{
		"trace bool",
		"if trace {",
	}

	for _, check := range checks {
		if !strings.Contains(code, check) {
			t.Errorf("generated code missing trace element: %q", check)
		}
	}
}

func TestValidatorNaming(t *testing.T) {
	gen := &Generator{}

	tests := []struct {
		title    string
		expected string
	}{
		{"always_true.script.spend", "AlwaysTrueScriptSpend"},
		{"always_true.script_no_params.else", "AlwaysTrueScriptNoParamsElse"},
		{"nested/sometimes_true.script.spend", "NestedSometimesTrueScriptSpend"},
		{"treasury.treasury.spend", "TreasuryTreasurySpend"},
		{"vendor.vendor.else", "VendorVendorElse"},
	}

	for _, tc := range tests {
		got := gen.validatorName(tc.title)
		if got != tc.expected {
			t.Errorf("validatorName(%q) = %q, want %q", tc.title, got, tc.expected)
		}
	}
}

func TestGeneratedCodeCompiles(t *testing.T) {
	// Skip if not running in an environment with Go compiler
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go compiler not found, skipping compilation test")
	}

	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "aiken2go_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate code for simple blueprint
	bp, err := LoadBlueprint("../../testdata/simple/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, nil, GeneratorOptions{PackageName: "contracts"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Write the generated code
	outFile := filepath.Join(tmpDir, "contracts.go")
	if err := os.WriteFile(outFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write generated code: %v", err)
	}

	// Create go.mod
	goMod := `module testmod

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Try to build
	cmd := exec.Command("go", "build", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("generated code failed to compile: %v\nOutput: %s\n\nGenerated code:\n%s", err, output, code)
	}
}

func TestGeneratedCodeCompiles_Complex(t *testing.T) {
	// Skip if not running in an environment with Go compiler
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go compiler not found, skipping compilation test")
	}

	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "aiken2go_test_complex")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate code for complex blueprint
	bp, err := LoadBlueprint("../../testdata/complex/plutus.json")
	if err != nil {
		t.Fatalf("failed to load blueprint: %v", err)
	}

	gen := NewGenerator(bp, nil, GeneratorOptions{PackageName: "treasury"})
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Write the generated code
	outFile := filepath.Join(tmpDir, "treasury.go")
	if err := os.WriteFile(outFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write generated code: %v", err)
	}

	// Create go.mod
	goMod := `module testmod

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Try to build
	cmd := exec.Command("go", "build", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("generated code failed to compile: %v\nOutput: %s\n\nGenerated code:\n%s", err, output, code)
	}
}
