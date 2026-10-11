package helpers

import "encoding/json"

func AnyToStruct[T any](m any) (*T, error) {
	var s T

	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &s)
	if err != nil {
		return nil, err
	}

	return &s, nil
}
