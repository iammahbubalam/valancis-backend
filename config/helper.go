package config

import (
	"fmt"
	"log"
	"os"
)

func getInt32Env(key string, fallback int32) int32 {
	if value, exists := os.LookupEnv(key); exists {
		if i, err := toInt32(value); err == nil {
			return i
		}
		log.Printf("Invalid int32 for %s, using fallback", key)
	}
	return fallback
}

func toInt32(s string) (int32, error) {
	// simple parsing
	var i int32
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
