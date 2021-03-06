package h

import "github.com/tidwall/gjson"

type JsonValue struct {
	value  string
	result *gjson.Result
}

func (j JsonValue) GetString(path string, defaultValue string) string {
	r := gjson.Get(j.value, path)
	if !r.Exists() {
		return defaultValue
	}
	return r.Str
}

func (j JsonValue) GetBool(path string, defaultValue bool) bool {
	r := gjson.Get(j.value, path)
	if !r.Exists() {
		return defaultValue
	}
	return r.Bool()
}
