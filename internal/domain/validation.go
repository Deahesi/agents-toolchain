package domain

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/invopop/validation"
)

const temperatureError = "agent.temperature must be a finite number between 0 and 2"

func requiredText(message string) validation.Rule {
	return validation.By(func(value any) error {
		if strings.TrimSpace(value.(string)) == "" {
			return errors.New(message)
		}
		return nil
	})
}

func localPath(message string) validation.Rule {
	return validation.By(func(value any) error {
		path := value.(string)
		if strings.TrimSpace(path) == "" || !filepath.IsLocal(path) {
			return errors.New(message)
		}
		return nil
	})
}

func validateModel(value any) error {
	name := value.(string)
	provider, model, ok := strings.Cut(name, "/")
	if !ok || strings.TrimSpace(provider) == "" || strings.TrimSpace(model) == "" ||
		strings.ContainsAny(name, " \t\r\n") {
		return errors.New("agent.model must have the form provider/model")
	}
	return nil
}

func validateFiniteNumber(value any) error {
	number := value.(float64)
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return errors.New(temperatureError)
	}
	return nil
}

func validateUniqueToolNames(value any) error {
	tools := value.([]Tool)
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		if names[tool.Name] {
			return fmt.Errorf("duplicate agent tool name %q", tool.Name)
		}
		names[tool.Name] = true
	}
	return nil
}
