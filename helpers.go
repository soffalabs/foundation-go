package sf

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

func GetBytes(key interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

