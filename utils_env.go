package soffa

import "os"

func GetEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if IsStrEmpty(value) {
		return fallback
	}
	return value
}
