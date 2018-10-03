package env

import "os"

// Get env variable. Use default value if the var. is not found
func Get(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

// Truthy will evaluate the env. var. value to bool
func Truthy(key string) bool {
	val := os.Getenv(key)
	return val == "true" || val == "1"
}
