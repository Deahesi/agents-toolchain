package config

import (
	"errors"
	"os"
)

func writeNewFile(root *os.Root, path string, content []byte) error {
	file, err := root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}

	_, writeErr := file.Write(content)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return errors.Join(err, root.Remove(path))
	}

	return nil
}
