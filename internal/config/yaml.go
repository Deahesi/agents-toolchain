package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

type validatable interface {
	Validate() error
}

func decode[T validatable](content []byte) (T, error) {
	var configuration T
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&configuration); err != nil {
		return configuration, fmt.Errorf("decode config: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return configuration, fmt.Errorf("decode trailing YAML: %w", err)
		}
		return configuration, errors.New("config must contain exactly one YAML document")
	}
	if err := configuration.Validate(); err != nil {
		return configuration, fmt.Errorf("invalid config: %w", err)
	}
	return configuration, nil
}

func encode(configuration validatable) ([]byte, error) {
	if err := configuration.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(configuration); err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("close YAML encoder: %w", err)
	}
	return buffer.Bytes(), nil
}
