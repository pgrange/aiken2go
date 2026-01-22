package blueprint

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/fxamacker/cbor/v2"
)

// PlutusData represents a Plutus Data value that can be serialized to CBOR.
// It corresponds to the on-chain data format used for datums and redeemers.
type PlutusData struct {
	// For constructor types
	Constr *ConstrPlutusData

	// For primitive types (only one should be set)
	Integer   *big.Int
	ByteString []byte
	List      []PlutusData
	Map       []PlutusDataMapEntry
}

// ConstrPlutusData represents a constructor with an index and fields.
type ConstrPlutusData struct {
	Index  uint64
	Fields []PlutusData
}

// PlutusDataMapEntry represents a key-value pair in a Plutus Data map.
type PlutusDataMapEntry struct {
	Key   PlutusData
	Value PlutusData
}

// CBOR tags for Plutus Data constructors
const (
	cborTagConstr0 = 121 // constructor index 0
	cborTagConstr1 = 122
	cborTagConstr2 = 123
	cborTagConstr3 = 124
	cborTagConstr4 = 125
	cborTagConstr5 = 126
	cborTagConstr6     = 127
	cborTagConstrBase  = 1280 // for index >= 7: tag = 1280 + index - 7
	cborTagBigPosInt   = 2
	cborTagBigNegInt   = 3
)

// NewConstrPlutusData creates a new constructor PlutusData.
func NewConstrPlutusData(index uint64, fields ...PlutusData) PlutusData {
	return PlutusData{
		Constr: &ConstrPlutusData{
			Index:  index,
			Fields: fields,
		},
	}
}

// NewIntPlutusData creates a new integer PlutusData.
func NewIntPlutusData(i *big.Int) PlutusData {
	return PlutusData{Integer: i}
}

// NewBytesPlutusData creates a new bytestring PlutusData.
func NewBytesPlutusData(b []byte) PlutusData {
	return PlutusData{ByteString: b}
}

// NewListPlutusData creates a new list PlutusData.
func NewListPlutusData(items ...PlutusData) PlutusData {
	return PlutusData{List: items}
}

// NewMapPlutusData creates a new map PlutusData.
func NewMapPlutusData(entries ...PlutusDataMapEntry) PlutusData {
	return PlutusData{Map: entries}
}

// MarshalCBOR serializes PlutusData to CBOR bytes using indefinite-length arrays
// to match Aiken's CBOR format.
func (p PlutusData) MarshalCBOR() ([]byte, error) {
	return p.toCBORBytes()
}

// toCBORBytes produces CBOR bytes with indefinite-length arrays (Aiken format).
func (p PlutusData) toCBORBytes() ([]byte, error) {
	em, err := cbor.EncOptions{BigIntConvert: cbor.BigIntConvertShortest}.EncMode()
	if err != nil {
		return nil, err
	}

	switch {
	case p.Constr != nil:
		var buf bytes.Buffer
		// Calculate tag
		var tag uint64
		if p.Constr.Index <= 6 {
			tag = cborTagConstr0 + p.Constr.Index
		} else {
			tag = cborTagConstrBase + p.Constr.Index - 7
		}
		// Write CBOR tag header (minimal encoding)
		if tag < 24 {
			buf.WriteByte(0xc0 + byte(tag))
		} else if tag < 256 {
			buf.WriteByte(0xd8)
			buf.WriteByte(byte(tag))
		} else {
			buf.WriteByte(0xd9)
			buf.WriteByte(byte(tag >> 8))
			buf.WriteByte(byte(tag))
		}
		// Empty arrays use definite-length, non-empty use indefinite
		if len(p.Constr.Fields) == 0 {
			buf.WriteByte(0x80) // empty array
		} else {
			buf.WriteByte(0x9f) // indefinite-length array start
			for _, f := range p.Constr.Fields {
				fieldBytes, err := f.toCBORBytes()
				if err != nil {
					return nil, err
				}
				buf.Write(fieldBytes)
			}
			buf.WriteByte(0xff) // break
		}
		return buf.Bytes(), nil

	case p.Integer != nil:
		return em.Marshal(p.Integer)

	case p.ByteString != nil:
		return em.Marshal(p.ByteString)

	case p.List != nil:
		var buf bytes.Buffer
		buf.WriteByte(0x9f) // indefinite-length array start
		for _, item := range p.List {
			itemBytes, err := item.toCBORBytes()
			if err != nil {
				return nil, err
			}
			buf.Write(itemBytes)
		}
		buf.WriteByte(0xff) // break
		return buf.Bytes(), nil

	case p.Map != nil:
		var buf bytes.Buffer
		buf.WriteByte(0xbf) // indefinite-length map start
		for _, entry := range p.Map {
			keyBytes, err := entry.Key.toCBORBytes()
			if err != nil {
				return nil, err
			}
			buf.Write(keyBytes)
			valBytes, err := entry.Value.toCBORBytes()
			if err != nil {
				return nil, err
			}
			buf.Write(valBytes)
		}
		buf.WriteByte(0xff) // break
		return buf.Bytes(), nil

	default:
		// Empty constructor 0
		return []byte{0xd8, 0x79, 0x9f, 0xff}, nil
	}
}

// UnmarshalCBOR deserializes PlutusData from CBOR bytes.
func (p *PlutusData) UnmarshalCBOR(data []byte) error {
	dm, err := cbor.DecOptions{
		BigIntDec: cbor.BigIntDecodePointer,
	}.DecMode()
	if err != nil {
		return err
	}

	var raw interface{}
	if err := dm.Unmarshal(data, &raw); err != nil {
		return err
	}

	result, err := fromCBORValue(raw)
	if err != nil {
		return err
	}
	*p = result
	return nil
}

// toCBORValue converts PlutusData to a CBOR-encodable value.
func (p PlutusData) toCBORValue() (interface{}, error) {
	switch {
	case p.Constr != nil:
		return p.constrToCBOR()
	case p.Integer != nil:
		return p.Integer, nil
	case p.ByteString != nil:
		return p.ByteString, nil
	case p.List != nil:
		items := make([]interface{}, len(p.List))
		for i, item := range p.List {
			v, err := item.toCBORValue()
			if err != nil {
				return nil, err
			}
			items[i] = v
		}
		return items, nil
	case p.Map != nil:
		// CBOR map preserving order
		m := make([]interface{}, 0, len(p.Map)*2)
		for _, entry := range p.Map {
			k, err := entry.Key.toCBORValue()
			if err != nil {
				return nil, err
			}
			v, err := entry.Value.toCBORValue()
			if err != nil {
				return nil, err
			}
			m = append(m, k, v)
		}
		// Use cbor.RawMessage to encode as map
		return encodeAsMap(p.Map)
	default:
		// Empty PlutusData - treat as empty constructor
		return cbor.Tag{Number: cborTagConstr0, Content: []interface{}{}}, nil
	}
}

func encodeAsMap(entries []PlutusDataMapEntry) (interface{}, error) {
	result := make(map[interface{}]interface{})
	for _, entry := range entries {
		k, err := entry.Key.toCBORValue()
		if err != nil {
			return nil, err
		}
		v, err := entry.Value.toCBORValue()
		if err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, nil
}

func (p PlutusData) constrToCBOR() (interface{}, error) {
	fields := make([]interface{}, len(p.Constr.Fields))
	for i, f := range p.Constr.Fields {
		v, err := f.toCBORValue()
		if err != nil {
			return nil, err
		}
		fields[i] = v
	}

	var tag uint64
	if p.Constr.Index <= 6 {
		tag = cborTagConstr0 + p.Constr.Index
	} else {
		tag = cborTagConstrBase + p.Constr.Index - 7
	}

	return cbor.Tag{Number: tag, Content: fields}, nil
}

// fromCBORValue converts a decoded CBOR value to PlutusData.
func fromCBORValue(v interface{}) (PlutusData, error) {
	switch val := v.(type) {
	case cbor.Tag:
		return fromCBORTag(val)
	case *big.Int:
		return PlutusData{Integer: val}, nil
	case int64:
		return PlutusData{Integer: big.NewInt(val)}, nil
	case uint64:
		return PlutusData{Integer: new(big.Int).SetUint64(val)}, nil
	case []byte:
		return PlutusData{ByteString: val}, nil
	case []interface{}:
		items := make([]PlutusData, len(val))
		for i, item := range val {
			pd, err := fromCBORValue(item)
			if err != nil {
				return PlutusData{}, err
			}
			items[i] = pd
		}
		return PlutusData{List: items}, nil
	case map[interface{}]interface{}:
		entries := make([]PlutusDataMapEntry, 0, len(val))
		for k, v := range val {
			key, err := fromCBORValue(k)
			if err != nil {
				return PlutusData{}, err
			}
			value, err := fromCBORValue(v)
			if err != nil {
				return PlutusData{}, err
			}
			entries = append(entries, PlutusDataMapEntry{Key: key, Value: value})
		}
		return PlutusData{Map: entries}, nil
	default:
		return PlutusData{}, fmt.Errorf("unsupported CBOR type: %T", v)
	}
}

func fromCBORTag(tag cbor.Tag) (PlutusData, error) {
	var index uint64
	switch {
	case tag.Number >= cborTagConstr0 && tag.Number <= cborTagConstr6:
		index = tag.Number - cborTagConstr0
	case tag.Number >= cborTagConstrBase:
		index = tag.Number - cborTagConstrBase + 7
	default:
		return PlutusData{}, fmt.Errorf("unsupported CBOR tag: %d", tag.Number)
	}

	content, ok := tag.Content.([]interface{})
	if !ok {
		return PlutusData{}, fmt.Errorf("constructor content is not an array")
	}

	fields := make([]PlutusData, len(content))
	for i, item := range content {
		pd, err := fromCBORValue(item)
		if err != nil {
			return PlutusData{}, err
		}
		fields[i] = pd
	}

	return PlutusData{
		Constr: &ConstrPlutusData{
			Index:  index,
			Fields: fields,
		},
	}, nil
}

// ToHex returns the CBOR encoding as a hex string.
func (p PlutusData) ToHex() (string, error) {
	data, err := p.MarshalCBOR()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", data), nil
}

// FromHex decodes PlutusData from a hex string.
func FromHex(hex string) (PlutusData, error) {
	data, err := hexDecode(hex)
	if err != nil {
		return PlutusData{}, err
	}
	var p PlutusData
	if err := p.UnmarshalCBOR(data); err != nil {
		return PlutusData{}, err
	}
	return p, nil
}

func hexDecode(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("hex string has odd length")
	}
	result := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		var b byte
		for j := 0; j < 2; j++ {
			c := s[i+j]
			switch {
			case c >= '0' && c <= '9':
				b = b*16 + (c - '0')
			case c >= 'a' && c <= 'f':
				b = b*16 + (c - 'a' + 10)
			case c >= 'A' && c <= 'F':
				b = b*16 + (c - 'A' + 10)
			default:
				return nil, fmt.Errorf("invalid hex character: %c", c)
			}
		}
		result[i/2] = b
	}
	return result, nil
}

// PlutusDataConverter is the interface that generated types implement.
type PlutusDataConverter interface {
	ToPlutusData() (PlutusData, error)
}

// PlutusDataDecoder is the interface for types that can be decoded from PlutusData.
type PlutusDataDecoder interface {
	FromPlutusData(PlutusData) error
}

// Marshal serializes a PlutusDataConverter to CBOR bytes.
func Marshal(v PlutusDataConverter) ([]byte, error) {
	pd, err := v.ToPlutusData()
	if err != nil {
		return nil, err
	}
	return pd.MarshalCBOR()
}

// Unmarshal deserializes CBOR bytes into a PlutusDataDecoder.
func Unmarshal(data []byte, v PlutusDataDecoder) error {
	var pd PlutusData
	if err := pd.UnmarshalCBOR(data); err != nil {
		return err
	}
	return v.FromPlutusData(pd)
}

// Equals compares two PlutusData values for equality.
func (p PlutusData) Equals(other PlutusData) bool {
	// Marshal both and compare bytes
	a, err := p.MarshalCBOR()
	if err != nil {
		return false
	}
	b, err := other.MarshalCBOR()
	if err != nil {
		return false
	}
	return bytes.Equal(a, b)
}
