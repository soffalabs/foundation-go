package soffa

import (
	"encoding/json"
)

func Convert(input interface{}, dest interface{}) error {
	bytes, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, &dest)
}
