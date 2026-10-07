package domain

import (
	"fmt"
	"regexp"
	"strings"
)

var agentNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

func ValidateAgentName(name string) error {
	if !agentNamePattern.MatchString(name) {
		return fmt.Errorf("invalid agent name %q: use 1–64 ASCII letters, digits, hyphens or underscores; start with a letter or digit", name)
	}
	switch strings.ToUpper(name) {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return fmt.Errorf("invalid agent name %q: reserved Windows device name", name)
	}
	return nil
}
