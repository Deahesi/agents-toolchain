package config

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"gopkg.in/yaml.v3"
)

func getAgentYAMLValue(content []byte, field string) (string, error) {
	parts := strings.Split(field, ".")
	for _, part := range parts {
		if part == "" || part != strings.TrimSpace(part) {
			return "", errors.New("field must be a dot-separated YAML path without empty components")
		}
	}
	// valueType, err := yamlFieldType(reflect.TypeFor[domain.AgentConfig](), parts)
	// if err != nil {
	// 	return nil, err
	// }
	var document yaml.Node
	if err := decodeDocument(content, &document); err != nil {
		return "", err
	}
	target, err := yamlPath(document.Content[0], parts)

	return target.Value, err

}

func setAgentYAMLValue(content []byte, field, value string) ([]byte, error) {
	parts := strings.Split(field, ".")
	for _, part := range parts {
		if part == "" || part != strings.TrimSpace(part) {
			return nil, errors.New("field must be a dot-separated YAML path without empty components")
		}
	}
	valueType, err := yamlFieldType(reflect.TypeFor[domain.AgentConfig](), parts)
	if err != nil {
		return nil, err
	}
	var document yaml.Node
	if err := decodeDocument(content, &document); err != nil {
		return nil, err
	}
	target, err := yamlPath(document.Content[0], parts)
	if err != nil {
		return nil, err
	}
	if target.Anchor != "" {
		return nil, errors.New("cannot replace an anchored field because other fields may reference it")
	}
	var replacement yaml.Node
	if indirectType(valueType).Kind() == reflect.String {
		replacement = yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
	} else {
		var input yaml.Node
		if err := decodeDocument([]byte(value), &input); err != nil {
			return nil, fmt.Errorf("parse value: %w", err)
		}
		replacement = *input.Content[0]
		if err := checkReplacementAliases(&replacement); err != nil {
			return nil, err
		}
		if err := checkYAMLValue(&replacement, valueType); err != nil {
			return nil, err
		}
	}
	replacement.HeadComment = target.HeadComment
	replacement.LineComment = target.LineComment
	replacement.FootComment = target.FootComment
	if target.Tag == replacement.Tag && target.Kind == yaml.ScalarNode {
		replacement.Style = target.Style
	}
	*target = replacement
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	encodeErr := encoder.Encode(&document)
	if err := errors.Join(encodeErr, encoder.Close()); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func indirectType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func yamlFieldType(t reflect.Type, parts []string) (reflect.Type, error) {
	for _, part := range parts {
		t = indirectType(t)
		switch t.Kind() {
		case reflect.Struct:
			var found bool
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				name, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
				if name == "" {
					name = strings.ToLower(field.Name)
				}
				if field.PkgPath == "" && name != "-" && name == part {
					t, found = field.Type, true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("unknown YAML field %q", part)
			}
		case reflect.Map:
			if t.Key().Kind() != reflect.String {
				return nil, errors.New("only maps with string keys are supported")
			}
			t = t.Elem()
		case reflect.Slice, reflect.Array:
			if index, err := strconv.Atoi(part); err != nil || index < 0 {
				return nil, fmt.Errorf("invalid list index %q", part)
			}
			t = t.Elem()
		case reflect.Interface:
			return t, nil // Tool-specific configuration has no static schema.
		default:
			return nil, fmt.Errorf("cannot traverse scalar field at %q", part)
		}
	}
	return t, nil
}

func yamlPath(node *yaml.Node, parts []string) (*yaml.Node, error) {
	for _, part := range parts {
		if node.Anchor != "" {
			return nil, errors.New("cannot traverse an anchored field because other fields may reference it")
		}
		if node.Tag == "!!null" {
			*node = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", HeadComment: node.HeadComment, LineComment: node.LineComment, FootComment: node.FootComment}
		}
		switch node.Kind {
		case yaml.MappingNode:
			var child *yaml.Node
			for i := 0; i < len(node.Content); i += 2 {
				if node.Content[i].Value == part {
					child = node.Content[i+1]
					break
				}
			}
			if child == nil {
				child = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null"}
				node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: part}, child)
			}
			node = child
		case yaml.SequenceNode:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(node.Content) {
				return nil, fmt.Errorf("list index %q is out of range", part)
			}
			node = node.Content[index]
		default:
			return nil, fmt.Errorf("cannot traverse YAML field %q (expected a mapping or list)", part)
		}
	}
	return node, nil
}

func checkReplacementAliases(node *yaml.Node) error {
	if node.Kind == yaml.AliasNode || node.Anchor != "" {
		return errors.New("YAML anchors and aliases are not supported in replacement values")
	}
	for _, child := range node.Content {
		if err := checkReplacementAliases(child); err != nil {
			return err
		}
	}
	return nil
}

// yaml.v3 accepts some coercions, including float-to-int truncation. Reject
// incompatible node types before decoding the complete config for validation.
func checkYAMLValue(node *yaml.Node, t reflect.Type) error {
	if node.Tag == "!!null" && (t.Kind() == reflect.Pointer || t.Kind() == reflect.Map || t.Kind() == reflect.Slice || t.Kind() == reflect.Interface) {
		return nil
	}
	t = indirectType(t)
	if node.Kind == yaml.AliasNode {
		return errors.New("YAML aliases are not supported in replacement values")
	}
	valid := false
	switch t.Kind() {
	case reflect.Interface:
		return nil
	case reflect.String:
		valid = node.Kind == yaml.ScalarNode && node.Tag == "!!str"
	case reflect.Bool:
		valid = node.Kind == yaml.ScalarNode && node.Tag == "!!bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		valid = node.Kind == yaml.ScalarNode && node.Tag == "!!int"
	case reflect.Float32, reflect.Float64:
		valid = node.Kind == yaml.ScalarNode && (node.Tag == "!!float" || node.Tag == "!!int")
	case reflect.Struct, reflect.Map:
		if node.Kind == yaml.MappingNode {
			for i := 0; i < len(node.Content); i += 2 {
				key := node.Content[i]
				if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
					return errors.New("mapping keys must be strings")
				}
				childType, err := yamlFieldType(t, []string{key.Value})
				if err != nil {
					return err
				}
				if err := checkYAMLValue(node.Content[i+1], childType); err != nil {
					return fmt.Errorf("field %q: %w", key.Value, err)
				}
			}
			return nil
		}
	case reflect.Slice, reflect.Array:
		if node.Kind == yaml.SequenceNode {
			for i, child := range node.Content {
				if err := checkYAMLValue(child, t.Elem()); err != nil {
					return fmt.Errorf("list item %d: %w", i, err)
				}
			}
			return nil
		}
	}
	if !valid {
		return fmt.Errorf("expected %s, got YAML %s", t, node.Tag)
	}
	return nil
}
