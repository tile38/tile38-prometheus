package common

import "os"

// GetEnv retrieves variables from the environment and falls back
// to a passed fallback variable if it isn't set
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
