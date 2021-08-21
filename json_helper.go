package sf

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
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
	data, err := json.Marshal(input)
	if err != nil {
		log.Warn(err)
		return ""
	}
	return string(data)
}

func FromJson(data string, dest interface{}) error {
	if err := json.Unmarshal([]byte(data), &dest); err != nil {
		return err
	}
	return nil
}