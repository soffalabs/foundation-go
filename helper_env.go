package sf

import "os"
import "strconv"


func Getenv(key string, fallback string, fallbackIf bool) string {
	value := os.Getenv(key)
	if IsStrEmpty(value) && fallbackIf {
		return fallback
	}
	return value
}

func Getenvi(key string, fallback int) int {
	value := os.Getenv(key)
	if IsStrEmpty(value)  {
		return fallback
	}
	iv, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return iv
}

func Getenvb(key string, fallback bool) bool {
	value := os.Getenv(key)
	if IsStrEmpty(value)  {
		return fallback
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return b
}
