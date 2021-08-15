package soffa

import (
	"encoding/json"
)

func ToJsonStr(input interface{}) (string, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ToJsonStrSafe(input interface{}) string {
	data, _ := json.Marshal(input)
	return string(data)
}

func FromJson(data string, dest interface{}) error {
	if err := json.Unmarshal([]byte(data), &dest); err != nil {
		return err
	}
	return nil
}