package repository

import "strings"

func maxInt(value, fallback int) int {
	if value < fallback {
		return fallback
	}
	return value
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
