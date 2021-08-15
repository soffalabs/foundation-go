package soffa

import "os"

func Getenv(key string, fallback string, fallbackIf bool) string {
	value := os.Getenv(key)
	if IsStrEmpty(value) && fallbackIf {
		return fallback
	}
	return value
}
