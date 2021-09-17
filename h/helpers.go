package h

import (
	"bytes"
	"encoding/gob"
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/log"
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

func Nil(data interface{}) interface{} {
	if IsNil(data) {
		return nil
	}
	return data
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
	switch data.(type) {
	case []byte:
		return data.([]byte), nil
	}
	if IsNil(data) {
		return nil, nil
	}
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(data)
	if err != nil {
		return nil, errors.Wrap(err, "bytes encoding failed")
	}
	return b.Bytes(), nil
}

func DecodeBytes(data interface{}, dest interface{}) error {
	if IsNil(dest) {
		return errors.New("unable to decode bytes into nul reference")
	}
	b, err := GetBytes(data)
	if err != nil {
		return err
	}
	if data == nil || len(b) == 0 {
		return nil
	}
	buf := bytes.NewBuffer(b)
	dev := gob.NewDecoder(buf)
	err = dev.Decode(dest)
	if err != nil {
		log.Default.Error(err)
		return errors.Wrap(err, "bytes decoding failed")
	}
	return nil
}
