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
