package b64lib

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strings"
)

var alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
var reverseAlphabet = []byte{0x3e, 0xff, 0xff, 0xff, 0x3f, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30, 0x31, 0x32, 0x33}

func Encode(b []byte) string {
	extra := len(b) % 3

	numSections := int(math.Ceil(float64(len(b)) / 3.0))
	encodedLen := numSections * 4
	result := make([]byte, encodedLen)

	var lastSectionStart int
	numFullSections := numSections
	if extra != 0 {
		numFullSections--
		lastSectionStart = numFullSections * 4
	}

	for section := 0; section < numFullSections; section++ {
		sin := section * 3
		sout := section * 4
		to4char(b[sin:sin+3], result[sout:sout+4])
	}

	if extra == 1 {
		// One extra byte out of three, so two padding bytes
		to4char(
			[]byte{b[len(b)-1], 0, 0},
			result[lastSectionStart:lastSectionStart+4],
		)
		result[lastSectionStart+2] = '='
		result[lastSectionStart+3] = '='
	} else if extra == 2 {
		to4char(
			[]byte{b[len(b)-2], b[len(b)-1], 0},
			result[lastSectionStart:lastSectionStart+4],
		)
		result[lastSectionStart+3] = '='
	}
	return string(result)
}

func Decode(s string) (out []byte, err error) {
	defer func() {
		// Turn panic into error, save on bounds checking
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("%v", r))
		}
	}()
	s = strings.Map(removeNewlinesMapper, s)
	b := []byte(s)
	var result bytes.Buffer
	for i := 0; i < len(b); i += 4 {
		result.Write(from4char(b[i : i+4]))
	}
	return result.Bytes(), nil
}

// []byte --> ABCD
func to4char(in []byte, out []byte) {
	// top six bits of byte 0
	k1 := in[0] >> 2
	out[0] = alphabet[k1]

	// bottom 2 bits of byte 0, top 4 bits of byte 1
	k2 := (in[0]&0x3)<<4 | (in[1]&0xF0)>>4
	out[1] = alphabet[k2]

	// bottom 4 bits of byte 1, top 2 bits of byte 2
	k3 := (in[1]&0xF)<<2 | (in[2]&0xC0)>>6
	out[2] = alphabet[k3]

	// bottom  6 bits of byte 2
	k4 := in[2] & 0x3F
	out[3] = alphabet[k4]

	// log.Printf("%b %b %b --> %b, %b, %b, %b", b[0], b[1], b[2], k1, k2, k3, k4)
}

// ABCD --> [3]byte
func from4char(b []byte) []byte {
	v1 := reverseAlphabet[b[0]-'+']
	v2 := reverseAlphabet[b[1]-'+']
	v3 := reverseAlphabet[b[2]-'+']
	v4 := reverseAlphabet[b[3]-'+']

	/*
		if v1 == 0xff {
			panic("Error byte 1")
		}
		if v2 == 0xff {
			panic("Error byte 2")
		}
		if v3 == 0xff {
			panic("Error byte 3")
		}
		if v4 == 0xff {
			panic("Error byte 4")
		}
	*/

	// ..XXXXXX ..XX0000 ..000000 ..000000
	b1 := v1<<2 | (v2&0x30)>>4
	// ..000000 ..00XXXX ..XXXX00 ..000000
	b2 := (v2&0xF)<<4 | (v3&0x3C)>>2
	// ..000000 ..000000 ..0000XX ..XXXXXX
	b3 := (v3&0x3)<<6 | v4

	//log.Printf("%b %b %b %b --> %b %b %b", v1, v2, v3, v4, b1, b2, b3)

	if b[2] == '=' && b[3] == '=' {
		return []byte{b1}
	} else if b[3] == '=' {
		return []byte{b1, b2}
	}
	return []byte{b1, b2, b3}
}

var removeNewlinesMapper = func(r rune) rune {
	if r == '\r' || r == '\n' {
		return -1
	}
	return r
}
