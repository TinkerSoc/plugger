package plugger

import (
	"fmt"
	"io"
)

// readExactly is a helper function which checks that the number of bytes read
// from an io.Reader is the same as the length of the buffer being read to.
func readExactly(p []byte, r io.Reader) error {
	n, err := r.Read(p)

	if n != len(p) {
		err = fmt.Errorf("Incorrect number of bytes. Expected %d, got %d.", len(p), n)
	}

	return err
}
