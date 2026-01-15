package blueprint

import (
	"math/big"
	"testing"
)

func TestPlutusData_Integer(t *testing.T) {
	// Test positive integer
	pd := NewIntPlutusData(big.NewInt(42))
	data, err := pd.MarshalCBOR()
	if err != nil {
		t.Fatalf("failed to marshal integer: %v", err)
	}

	var decoded PlutusData
	if err := decoded.UnmarshalCBOR(data); err != nil {
		t.Fatalf("failed to unmarshal integer: %v", err)
	}

	if decoded.Integer == nil || decoded.Integer.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("expected 42, got %v", decoded.Integer)
	}
}

func TestPlutusData_ByteString(t *testing.T) {
	pd := NewBytesPlutusData([]byte{0xde, 0xad, 0xbe, 0xef})
	data, err := pd.MarshalCBOR()
	if err != nil {
		t.Fatalf("failed to marshal bytes: %v", err)
	}

	var decoded PlutusData
	if err := decoded.UnmarshalCBOR(data); err != nil {
		t.Fatalf("failed to unmarshal bytes: %v", err)
	}

	if string(decoded.ByteString) != string([]byte{0xde, 0xad, 0xbe, 0xef}) {
		t.Errorf("expected deadbeef, got %x", decoded.ByteString)
	}
}

func TestPlutusData_List(t *testing.T) {
	pd := NewListPlutusData(
		NewIntPlutusData(big.NewInt(1)),
		NewIntPlutusData(big.NewInt(2)),
		NewIntPlutusData(big.NewInt(3)),
	)
	data, err := pd.MarshalCBOR()
	if err != nil {
		t.Fatalf("failed to marshal list: %v", err)
	}

	var decoded PlutusData
	if err := decoded.UnmarshalCBOR(data); err != nil {
		t.Fatalf("failed to unmarshal list: %v", err)
	}

	if len(decoded.List) != 3 {
		t.Errorf("expected 3 items, got %d", len(decoded.List))
	}
}

func TestPlutusData_Constructor(t *testing.T) {
	// Test constructor index 0 (tag 121)
	pd := NewConstrPlutusData(0, NewIntPlutusData(big.NewInt(42)))
	data, err := pd.MarshalCBOR()
	if err != nil {
		t.Fatalf("failed to marshal constructor: %v", err)
	}

	var decoded PlutusData
	if err := decoded.UnmarshalCBOR(data); err != nil {
		t.Fatalf("failed to unmarshal constructor: %v", err)
	}

	if decoded.Constr == nil {
		t.Fatal("expected constructor")
	}
	if decoded.Constr.Index != 0 {
		t.Errorf("expected index 0, got %d", decoded.Constr.Index)
	}
	if len(decoded.Constr.Fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(decoded.Constr.Fields))
	}
}

func TestPlutusData_Constructor_HighIndex(t *testing.T) {
	// Test constructor index 7+ (tag 1280+)
	pd := NewConstrPlutusData(10)
	data, err := pd.MarshalCBOR()
	if err != nil {
		t.Fatalf("failed to marshal constructor: %v", err)
	}

	var decoded PlutusData
	if err := decoded.UnmarshalCBOR(data); err != nil {
		t.Fatalf("failed to unmarshal constructor: %v", err)
	}

	if decoded.Constr == nil {
		t.Fatal("expected constructor")
	}
	if decoded.Constr.Index != 10 {
		t.Errorf("expected index 10, got %d", decoded.Constr.Index)
	}
}

func TestPlutusData_Unit(t *testing.T) {
	// Unit/Void is constructor 0 with no fields
	pd := NewConstrPlutusData(0)

	hex, err := pd.ToHex()
	if err != nil {
		t.Fatalf("failed to marshal unit: %v", err)
	}
	// Constructor 0 with empty fields: d87980 (tag 121 + empty array)
	if hex != "d87980" {
		t.Errorf("expected d87980, got %s", hex)
	}
}

func TestPlutusData_Bool(t *testing.T) {
	// False = constructor 0, True = constructor 1
	falsePd := NewConstrPlutusData(0)
	truePd := NewConstrPlutusData(1)

	falseHex, _ := falsePd.ToHex()
	trueHex, _ := truePd.ToHex()

	if falseHex != "d87980" {
		t.Errorf("False: expected d87980, got %s", falseHex)
	}
	if trueHex != "d87a80" {
		t.Errorf("True: expected d87a80, got %s", trueHex)
	}
}

func TestPlutusData_FromHex(t *testing.T) {
	// Test decoding from known hex
	pd, err := FromHex("d87980") // Unit/False
	if err != nil {
		t.Fatalf("failed to decode hex: %v", err)
	}

	if pd.Constr == nil || pd.Constr.Index != 0 {
		t.Error("expected constructor 0")
	}
}

func TestPlutusData_RoundTrip(t *testing.T) {
	// Complex nested structure
	original := NewConstrPlutusData(0,
		NewIntPlutusData(big.NewInt(12345)),
		NewBytesPlutusData([]byte("hello")),
		NewListPlutusData(
			NewIntPlutusData(big.NewInt(1)),
			NewIntPlutusData(big.NewInt(2)),
		),
		NewConstrPlutusData(1,
			NewBytesPlutusData([]byte("nested")),
		),
	)

	data, err := original.MarshalCBOR()
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded PlutusData
	if err := decoded.UnmarshalCBOR(data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !original.Equals(decoded) {
		t.Error("round-trip failed: values don't match")
	}
}
