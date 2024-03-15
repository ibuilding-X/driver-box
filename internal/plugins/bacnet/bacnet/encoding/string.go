package encoding

import (
	"fmt"
	"golang.org/x/text/encoding/unicode"
)

type stringType uint8

// Supported String btypes
const (
	//https://github.com/stargieg/bacnet-stack/blob/master/include/bacenum.h#L1261
	stringUTF8    stringType = 0 //same as ANSI_X34
	characterUCS2 stringType = 4 //johnson controllers use this
)

func (e *Encoder) string(s string) {
	e.write(stringUTF8)
	e.write([]byte(s))
}

func decodeUCS2(s string) (string, error) {
	dec := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
	out, err := dec.String(s)
	if err != nil {
		return "", err
	}
	return out, err

}

func (d *Decoder) string(s *string, len int) error {
	var t stringType
	d.decode(&t)
	switch t {
	case stringUTF8:
	case characterUCS2:
	default:
		return fmt.Errorf("unsupported string format %d", t)
	}
	b := make([]byte, len)
	d.decode(b)

	if t == characterUCS2 {
		out, err := decodeUCS2(string(b))
		if err != nil {
			return fmt.Errorf("unable to decode string format characterUCS2%d", t)
		}
		*s = out
	} else {
		*s = string(b)
	}

	return d.Error()

}
func (e *Encoder) octetstring(b []byte) {
	e.write([]byte(b))
}
func (d *Decoder) octetstring(b *[]byte, len int) {
	*b = make([]byte, len)
	d.decode(b)
}
