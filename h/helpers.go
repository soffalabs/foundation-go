package h

import (
	"bytes"
	"encoding/gob"
	"github.com/soffa-io/soffa-core-go/errors"
	"reflect"
)

func AnyStr(candidates ...string) string {
	for _, s := range candidates {
		if !IsStrEmpty(s) {
			return s
		}
	}
	return ""
}

func IsNil(data interface{}) bool {
	if data == nil {
		return true
	}
	t := reflect.ValueOf(data)
	if t.Kind() == reflect.Invalid {
		return true
	}
	if t.Kind() == reflect.Ptr && t.IsNil() {
		return true
	}
	return false

}

func GetBytes(data interface{}) ([]byte, error) {
	if IsNil(data) {
		return nil, nil
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, errors.Wrap(err, "bytes encoding failed")
	}
	return buf.Bytes(), nil
}

func DecodeBytes(data []byte, dest interface{}) error {
	if IsNil(dest) {
		return errors.New("unable to decode bytes into nul reference")
	}
	if data == nil || len(data)==0 {
		return nil
	}
	buf := bytes.NewBuffer(data)
	dev := gob.NewDecoder(buf)
	err := dev.Decode(dest)
	if err != nil {
		return errors.Wrap(err, "bytes decoding failed")
	}
	return nil
}

