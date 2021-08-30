package h

import (
	"encoding/json"
	"fmt"
	"github.com/soffa-io/soffa-core-go/log"
)

func ToBytesSafe(input interface{}) []byte {
	if input == nil {
		return nil
	}
	return []byte(ToJsonStrSafe(input))
}

func ToJson(input interface{}) ([]byte, error) {
	if input == nil {
		return nil, nil
	}
	switch input.(type) {
	case string:
		return []byte(input.(string)), nil
	default:
		data, err := json.Marshal(input)
		if err != nil {
			log.Default.Errorf("marshaling of %s failed with error %v", data, err)
			return nil, err
		}
		fmt.Printf("%s\n", data)
		return data, nil
	}
}

func Convert(input interface{}, dest interface{}) error {
	b, err := ToJson(input)
	if err != nil {
		return nil
	}
	return FromJson(b, dest)
}

func ToJsonStr(input interface{}) (string, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ToJsonStrSafe(input interface{}) string {
	if IsNil(input) {
		return ""
	}
	data, err := json.Marshal(input)
	if err != nil {
		log.Default.Fatal(err)
	}
	return string(data)
}

func FromJson(data []byte, dest interface{}) error {
	if err := json.Unmarshal(data, &dest); err != nil {
		return err
	}
	return nil
}

func FromJsonStr(data string, dest interface{}) error {
	return FromJson([]byte(data), dest)
}