package commons

import (
	"bytes"
	"encoding/gob"
)

func AnyStr(candidates ...string) string {
	for _, s := range candidates {
		if !IsStrEmpty(s) {
			return s
		}
	}
	return ""
}

func A(err error, fn func() error) error {
	if err != nil {
		return err
	}
	return fn()
}

func GetBytes(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeBytes(data []byte, dest interface{}) error {
	buf := bytes.NewBuffer(data)
	dev := gob.NewDecoder(buf)
	return dev.Decode(dest)
}

