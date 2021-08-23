package soffa_core

import (
	"encoding/json"
	"github.com/soffa-io/soffa-core-go/log"
)

func ToBytesSafe(input interface{}) []byte {
	return []byte(ToJsonStrSafe(input))
}

func ToJson(input interface{}) ([]byte, error) {
	return json.Marshal(input)
}

func ToJsonStr(input interface{}) (string, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ToJsonStrSafe(input interface{}) string {
	data, err := json.Marshal(input)
	if err != nil {
		log.Warn(err)
		return ""
	}
	return string(data)
}

func FromJson(data []byte, dest interface{}) error {
	if err := json.Unmarshal(data, &dest); err != nil {
		return err
	}
	return nil
}

func FromJsonStrng(data string, dest interface{}) error {
	return FromJson([]byte(data), dest)
}