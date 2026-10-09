package domain

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/invopop/validation"
)

func requiredText(message string) validation.Rule {
	return validation.By(func(value any) error {
		value, isNil := validation.Indirect(value)
		if isNil {
			return errors.New(message)
		}
		text, err := validation.EnsureString(value)
		if err != nil || strings.TrimSpace(text) == "" {
			return errors.New(message)
		}
		return nil
	})
}

func localPath(message string) validation.Rule {
	return validation.By(func(value any) error {
		value, isNil := validation.Indirect(value)
		if isNil {
			return errors.New(message)
		}
		path, err := validation.EnsureString(value)
		if err != nil || strings.TrimSpace(path) == "" || !filepath.IsLocal(path) {
			return errors.New(message)
		}
		return nil
	})
}

func existingDirectory(message string) validation.Rule {
	return validation.By(func(value any) error {
		value, isNil := validation.Indirect(value)
		if isNil {
			return nil
		}
		path, err := validation.EnsureString(value)
		if err != nil {
			return errors.New(message)
		}
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("%s: %w", message, err)
		}
		if !info.IsDir() {
			return errors.New(message)
		}
		return nil
	})
}

func oneOf[T any](message string, values ...T) validation.Rule {
	allowed := make([]any, len(values))
	for i, value := range values {
		allowed[i] = value
	}
	return validation.In(allowed...).Error(message)
}

func withoutWhitespace(message string) validation.Rule {
	return validation.By(func(value any) error {
		value, isNil := validation.Indirect(value)
		if isNil {
			return nil
		}
		text, err := validation.EnsureString(value)
		if err != nil || strings.ContainsAny(text, " \t\r\n") {
			return errors.New(message)
		}
		return nil
	})
}

func finiteNumber(message string) validation.Rule {
	return validation.By(func(value any) error {
		value, isNil := validation.Indirect(value)
		if isNil {
			return nil
		}
		if _, err := validation.ToInt(value); err == nil {
			return nil
		}
		if _, err := validation.ToUint(value); err == nil {
			return nil
		}
		number, err := validation.ToFloat(value)
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return errors.New(message)
		}
		return nil
	})
}

func positiveInteger(message string) validation.Rule {
	return validation.By(func(value any) error {
		value, isNil := validation.Indirect(value)
		if isNil {
			return nil
		}
		if number, err := validation.ToInt(value); err == nil && number > 0 {
			return nil
		}
		if number, err := validation.ToUint(value); err == nil && number > 0 {
			return nil
		}
		return errors.New(message)
	})
}

func uniqueBy[T any, K comparable](message string, key func(T) K) validation.Rule {
	return validation.By(func(value any) error {
		value, isNil := validation.Indirect(value)
		if isNil {
			return nil
		}
		items, ok := value.([]T)
		if !ok {
			return errors.New(message)
		}
		seen := make(map[K]bool, len(items))
		for _, item := range items {
			itemKey := key(item)
			if seen[itemKey] {
				return fmt.Errorf("%s: %v", message, itemKey)
			}
			seen[itemKey] = true
		}
		return nil
	})
}
