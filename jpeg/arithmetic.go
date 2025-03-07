// Copyright 2025 Robert Ancell. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jpeg

import (
	"io"
)

// arithmetic is a Arithmetic decoder, specified in section D.
type arithmetic struct {
	// Conditioning value.
	conditioning uint8

	a uint16

	c uint16

	// last read byte.
	d uint8

	// number of bits FIXME.
	ct uint8

	dcNonZero [5]arithmeticState
	dcSign    [5]arithmeticState
	dcSp      [5]arithmeticState
	dcSn      [5]arithmeticState
	dcXStates [15]arithmeticState
	dcMStates [14]arithmeticState

	acEndOfBlock  [63]arithmeticState
	acNonZero     [63]arithmeticState
	acSnSpX1      [63]arithmeticState
	acLowXStates  [14]arithmeticState
	acHighXStates [14]arithmeticState
	acLowMStates  [14]arithmeticState
	acHighMStates [14]arithmeticState
}

type arithmeticState struct {
	index uint8
	mps   bool
}

// Qe values and probability estimation state machine from table D.3.
var arithmeticStateMachine = []struct {
	qe        uint16
	nextLps   uint8
	nextMps   uint8
	switchMps bool
}{
	{0x5a1d, 1, 1, true},
	{0x2586, 14, 2, false},
	{0x1114, 16, 3, false},
	{0x080B, 18, 4, false},
	{0x03D8, 20, 5, false},
	{0x01DA, 23, 6, false},
	{0x00E5, 25, 7, false},
	{0x006F, 28, 8, false},
	{0x0036, 30, 9, false},
	{0x001A, 33, 10, false},
	{0x000D, 35, 11, false},
	{0x0006, 9, 12, false},
	{0x0003, 10, 13, false},
	{0x0001, 12, 13, false},
	{0x5A7F, 15, 15, true},
	{0x3F25, 36, 16, false},
	{0x2CF2, 38, 17, false},
	{0x207C, 39, 18, false},
	{0x17B9, 40, 19, false},
	{0x1182, 42, 20, false},
	{0x0CEF, 43, 21, false},
	{0x09A1, 45, 22, false},
	{0x072F, 46, 23, false},
	{0x055C, 48, 24, false},
	{0x0406, 49, 25, false},
	{0x0303, 51, 26, false},
	{0x0240, 52, 27, false},
	{0x01B1, 54, 28, false},
	{0x0144, 56, 29, false},
	{0x00F5, 57, 30, false},
	{0x00B7, 59, 31, false},
	{0x008A, 60, 32, false},
	{0x0068, 62, 33, false},
	{0x004E, 63, 34, false},
	{0x003B, 32, 35, false},
	{0x002C, 33, 9, false},
	{0x5AE1, 37, 37, true},
	{0x484C, 64, 38, false},
	{0x3A0D, 65, 39, false},
	{0x2EF1, 67, 40, false},
	{0x261F, 68, 41, false},
	{0x1F33, 69, 42, false},
	{0x19A8, 70, 43, false},
	{0x1518, 72, 44, false},
	{0x1177, 73, 45, false},
	{0x0E74, 74, 46, false},
	{0x0BFB, 75, 47, false},
	{0x09F8, 77, 48, false},
	{0x0861, 78, 49, false},
	{0x0706, 79, 50, false},
	{0x05CD, 48, 51, false},
	{0x04DE, 50, 52, false},
	{0x040F, 50, 53, false},
	{0x0363, 51, 54, false},
	{0x02D4, 52, 55, false},
	{0x025C, 53, 56, false},
	{0x01F8, 54, 57, false},
	{0x01A4, 55, 58, false},
	{0x0160, 56, 59, false},
	{0x0125, 57, 60, false},
	{0x00F6, 58, 61, false},
	{0x00CB, 59, 62, false},
	{0x00AB, 61, 63, false},
	{0x008F, 61, 32, false},
	{0x5B12, 65, 65, true},
	{0x4D04, 80, 66, false},
	{0x412C, 81, 67, false},
	{0x37D8, 82, 68, false},
	{0x2FE8, 83, 69, false},
	{0x293C, 84, 70, false},
	{0x2379, 86, 71, false},
	{0x1EDF, 87, 72, false},
	{0x1AA9, 87, 73, false},
	{0x174E, 72, 74, false},
	{0x1424, 72, 75, false},
	{0x119C, 74, 76, false},
	{0x0F6B, 74, 77, false},
	{0x0D51, 75, 78, false},
	{0x0BB6, 77, 79, false},
	{0x0A40, 77, 48, false},
	{0x5832, 80, 81, true},
	{0x4D1C, 88, 82, false},
	{0x438E, 89, 83, false},
	{0x3BDD, 90, 84, false},
	{0x34EE, 91, 85, false},
	{0x2EAE, 92, 86, false},
	{0x299A, 93, 87, false},
	{0x2516, 86, 71, false},
	{0x5570, 88, 89, true},
	{0x4CA9, 95, 90, false},
	{0x44D9, 96, 91, false},
	{0x3E22, 97, 92, false},
	{0x3824, 99, 93, false},
	{0x32B4, 99, 94, false},
	{0x2E17, 93, 86, false},
	{0x56A8, 95, 96, true},
	{0x4F46, 101, 97, false},
	{0x47E5, 102, 98, false},
	{0x41CF, 103, 99, false},
	{0x3C3D, 104, 100, false},
	{0x375E, 99, 93, false},
	{0x5231, 105, 102, false},
	{0x4C0F, 106, 103, false},
	{0x4639, 107, 104, false},
	{0x415E, 103, 99, false},
	{0x5627, 105, 106, true},
	{0x50E7, 108, 107, false},
	{0x4B85, 109, 103, false},
	{0x5597, 110, 109, false},
	{0x504F, 111, 107, false},
	{0x5A10, 110, 111, true},
	{0x5522, 112, 109, false},
	{0x59EB, 112, 111, true},
}

// processDAC processes a Define Arithmetic Coding Conditioning marker, and initializes an arithmetic
// struct from its contents. Specified in section B.2.4.3.
func (d *decoder) processDAC(n int) error {
	for n > 0 {
		if n < 2 {
			return FormatError("DAC has wrong length")
		}
		if err := d.readFull(d.tmp[:2]); err != nil {
			return err
		}
		tc := d.tmp[0] >> 4
		if tc > maxTc {
			return FormatError("bad Tc value")
		}
		tb := d.tmp[0] & 0x0f
		if tb > maxTb {
			return FormatError("bad Tb value")
		}
		a := &d.arith[tc][tb]
		cs := d.tmp[1]
		if tc == 1 && (cs < 1 || cs > 63) {
			return FormatError("bad Cs value")
		}
		a.conditioning = cs
		n -= 2
	}
	return nil
}

// decodeArithmeticDC returns the next Arithmetic-coded DC value from the bit-stream,
// decoded according to a.
func (d *decoder) decodeArithmeticDC(a *arithmetic) (int32, error) {
	bit, err := d.decodeArithmeticBit(a, &a.dcNonZero[0])
	if err != nil {
		return 0, err
	}
	if !bit {
		return 0, nil
	}

	bit, err = d.decodeArithmeticBit(a, &a.dcSign[0])
	if err != nil {
		return 0, err
	}
	var sign int32 = 0
	var magState *arithmeticState
	if bit {
		sign = -1
		magState = &a.dcSn[0]
	} else {
		sign = 1
		magState = &a.dcSp[0]
	}

	bit, err = d.decodeArithmeticBit(a, magState)
	if err != nil {
		return 0, err
	}
	if !bit {
		return sign, nil
	}

	var width = 1
	for {
		bit, err = d.decodeArithmeticBit(a, &a.dcXStates[width])
		if err != nil {
			return 0, err
		}
		if !bit {
			break
		}
		width += 1
	}

	var magnitude int32 = 1
	for _ = range width - 1 {
		bit, err = d.decodeArithmeticBit(a, &a.dcMStates[width-2])
		if err != nil {
			return 0, err
		}
		magnitude <<= 1
		if bit {
			magnitude |= 1
		}
	}
	magnitude += 1

	return sign * magnitude, nil
}

// decodeArithmeticAC returns the next Arithmetic-coded AC value from the bit-stream,
// decoded according to a.
func (d *decoder) decodeArithmeticAC(a *arithmetic) (int32, error) {
	var fixedState arithmeticState
	bit, err := d.decodeArithmeticBit(a, &fixedState)
	if err != nil {
		return 0, err
	}
	var sign int32 = 0
	if bit {
		sign = -1
	} else {
		sign = 1
	}

	bit, err = d.decodeArithmeticBit(a, &a.acSnSpX1[0])
	if err != nil {
		return 0, err
	}
	if !bit {
		return sign, nil
	}

	var width = 1
	bit, err = d.decodeArithmeticBit(a, &a.acSnSpX1[0])
	if err != nil {
		return 0, err
	}
	if bit {
		width += 1
		for {
			bit, err = d.decodeArithmeticBit(a, &a.acLowXStates[width-2])
			if err != nil {
				return 0, err
			}
			if !bit {
				break
			}
			width += 1
		}
	}

	var magnitude int32 = 1
	for _ = range width - 1 {
		bit, err = d.decodeArithmeticBit(a, &a.acLowMStates[width+2])
		if err != nil {
			return 0, err
		}
		magnitude <<= 1
		if bit {
			magnitude |= 1
		}
	}
	magnitude += 1

	return sign * magnitude, nil
}

func (d *decoder) decodeArithmeticBit(a *arithmetic, state *arithmeticState) (bool, error) {
	a.a -= arithmeticStateMachine[state.index].qe
	var bit bool = false
	if a.c < a.a {
		if a.a < 0x8000 {
			bit = a.condMpsExchange(state)
			err := a.renormalize(d)
			if err != nil {
				return false, err
			}
		} else {
			bit = state.mps
		}
	} else {
		bit = a.condLpsExchange(state)
		err := a.renormalize(d)
		if err != nil {
			return false, err
		}
	}

	return bit, nil
}

func (a *arithmetic) condMpsExchange(state *arithmeticState) bool {
	s := arithmeticStateMachine[state.index]
	var bit bool = false
	if a.a < s.qe {
		bit = !state.mps
		if s.switchMps {
			state.mps = !state.mps
		}
		state.index = s.nextLps
	} else {
		bit = state.mps
		state.index = s.nextMps
	}
	return bit
}

func (a *arithmetic) condLpsExchange(state *arithmeticState) bool {
	s := arithmeticStateMachine[state.index]
	var bit bool = false
	if a.a < s.qe {
		bit = state.mps
		state.index = s.nextMps
	} else {
		bit = !state.mps
		if s.switchMps {
			state.mps = !state.mps
		}
		state.index = s.nextLps
	}
	a.a = s.qe
	return bit
}

func (a *arithmetic) renormalize(d *decoder) error {
	for {
		if a.ct == 16 {
			err := a.byteIn(d)
			if err != nil {
				return err
			}
		}
		a.a <<= 1
		a.c = (a.c << 1) | (uint16(a.d) >> 7)
		a.d <<= 1
		if a.ct == 0 {
			return nil
		}
		a.ct -= 1
		if a.a >= 0x8000 {
			return nil
		}
	}
}

func (a *arithmetic) byteIn(d *decoder) error {
	c, err := d.readByteStuffedByte()
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			a.d = 0
			a.ct += 8
			return nil
		}
		return err
	}

	a.d = c
	a.ct += 8
	return nil
}
