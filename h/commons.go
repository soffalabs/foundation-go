package h

import (
	"strings"
)

var (
	nilInteface interface{}
)

func IsNotEmpty(value interface{}) bool {
	return !IsEmpty(value)
}
func IsEmpty(value interface{}) bool {
	if IsNil(value) {
		return true
	}
	switch value.(type) {
	case string:
		return len(strings.TrimSpace(value.(string))) == 0
	case *string:
		unwrapped := value.(*string)
		if unwrapped == nil {
			return true
		}
		return len(strings.TrimSpace(*unwrapped)) == 0
	case []interface{}:
		return len(value.([]interface{})) == 0
	default:
		match := value == nilInteface
		return match
	}
}

func IsAllEmpty(values ...interface{}) bool {
	for _, v := range values {
		if !IsEmpty(v) {
			return false
		}
	}
	return true
}
