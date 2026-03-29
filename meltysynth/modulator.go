package meltysynth

import (
	"errors"
	"io"
)

// Since modulators will not be supported, we discard the data.
func discardModulatorData(r io.Reader, size int32) error {
	if size%10 != 0 {
		return errors.New("the modulator list is invalid")
	}

	if _, err := io.ReadFull(r, make([]byte, size)); err != nil {
		return err
	}

	return nil
}
