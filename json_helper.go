package sf

import (
	"encoding/json"
)

func ToBytesSafe(input interface{}) []byte {
	return []byte(ToJsonStrSafe(input))
}

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