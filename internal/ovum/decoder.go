// Package ovum implements the Ovum heat-pump register decoding protocol.
//
// Each logical value is stored in a block of 10 consecutive Modbus holding
// registers. Layout (mirrors Python ovum.py):
//
//	reg 0      - value low word  (forms a 32-bit int together with reg1)
//	reg 1      - value high word
//	reg 2      - min value
//	reg 3      - max value
//	reg 4 [15:12] precision (0-15 decimal places)
//	      [6:0]  unit ID
//	reg 5 [15]   0 = data register, 1 = menu item
//	      [14]   0 = read-only
//	reg 6      - parameter chars 1 & 2  (e.g. 'R','p')
//	reg 7      - parameter chars 3 & 4  (e.g. 's',' ')
//	reg 8      - descriptor ID
//	reg 9      - multi/enum type ID (0 = plain numeric)
package ovum

import (
	"encoding/binary"
	"fmt"
	"math"
)

// RegisterBlockSize is the number of holding registers per Ovum value.
const RegisterBlockSize = 10

// RegisterValue is the fully decoded result for one Ovum register block.
type RegisterValue struct {
	Address    uint16
	Parameter  string // 4-char identifier, e.g. "Rps "
	Value      float64
	Precision  int
	UnitID     int
	IsReadOnly bool
	IsMenu     bool
	IsEnum     bool // multi_id != 0 means enum/text value, not a plain float
}

// Decode interprets the 20 raw bytes (10 x 2-byte big-endian registers) that
// were read from the heat pump and returns a RegisterValue. Returns an error
// when the block is a menu item or an enum value that cannot be a float64.
func Decode(address uint16, data []byte) (RegisterValue, error) {
	if len(data) < RegisterBlockSize*2 {
		return RegisterValue{}, fmt.Errorf("need %d bytes, got %d", RegisterBlockSize*2, len(data))
	}

	reg := func(i int) uint16 {
		return binary.BigEndian.Uint16(data[i*2:])
	}
	regBit := func(i, bit int) bool {
		return (reg(i)>>uint(bit))&1 == 1
	}

	// reg5 bit15 == 1 means this block is a menu header, not a data register
	isMenu := regBit(5, 15)
	isReadonly := !regBit(5, 14)

	parameter := string([]byte{
		byteChar(reg(6) >> 8),
		byteChar(reg(6) & 0xFF),
		byteChar(reg(7) >> 8),
		byteChar(reg(7) & 0xFF),
	})

	rv := RegisterValue{
		Address:    address,
		Parameter:  parameter,
		IsMenu:     isMenu,
		IsReadOnly: isReadonly,
	}

	if isMenu {
		return rv, fmt.Errorf("register 0x%04x is a menu item", address)
	}

	// 32-bit signed value: high word = reg1, low word = reg0
	raw32 := uint32(reg(1))<<16 | uint32(reg(0))
	var signed int32
	if raw32&0x80000000 != 0 {
		signed = -int32((raw32 ^ 0xFFFFFFFF) + 1)
	} else {
		signed = int32(raw32)
	}

	// Precision: top 4 bits of reg4 (bits 15-12)
	reg4 := reg(4)
	precision := int(reg4 >> 12)
	unitID := int(reg4 & 0x7F)

	// reg9 != 0 means this is a text/enum value - cannot publish as float
	if reg(9) != 0 {
		rv.IsEnum = true
		rv.Precision = precision
		rv.UnitID = unitID
		return rv, fmt.Errorf("register 0x%04x (%s) is an enum/text value", address, parameter)
	}

	factor := math.Pow10(precision)
	valueFloat := math.Round(float64(signed)*math.Pow10(-precision)*factor) / factor

	rv.Value = valueFloat
	rv.Precision = precision
	rv.UnitID = unitID
	return rv, nil
}

// byteChar returns the byte as printable ASCII, or a space for non-printable.
func byteChar(b uint16) byte {
	if b > 31 && b < 127 {
		return byte(b)
	}
	return ' '
}
