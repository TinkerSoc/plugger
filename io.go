package plugger

import (
	"fmt"
	"io"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// LabelEncoding is the UTF-16 little-endian encoding used in the RDT format.
var LabelEncoding = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)

// LabelDecoder cleans incoming labels to strip trailing null bytes.
var LabelDecoder = transform.Chain(LabelEncoding.NewDecoder(), transform.RemoveFunc(nullBytes))

// readExactly is a helper function which checks that the number of bytes read
// from an io.Reader is the same as the length of the buffer being read to.
func readExactly(p []byte, r io.Reader) error {
	n, err := r.Read(p)

	if n != len(p) {
		err = fmt.Errorf("Incorrect number of bytes. Expected %d, got %d.", len(p), n)
	}

	return err
}

// nullBytes returns true if the provided rune is a null byte.
func nullBytes(r rune) bool {
	return r == 0
}
