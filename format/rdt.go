package format

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"

	"github.com/TinkerSoc/plugger"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// RDTByteOrder is the byte order for RDT files.
var RDTByteOrder = binary.LittleEndian

// RDTNameCodec is the encoding used by the 16-character UTF16 Labels in RDT.
var RDTNameCodec = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)

// RDTNameDecoder is the same as the RDTLabelCodec, but strips incoming null bytes.
var RDTNameDecoder = transform.Chain(RDTNameCodec.NewDecoder(), transform.RemoveFunc(isNullByte))

// RDTPlug is the raw file format for MD-380 family radios including the RT-3.
// The magic numbers are concentrated here for easy reference. The binary RDT
// file can be read directly into this struct provided the correct byte order
// is used (see RDTByteOrder).
type RDTPlug struct {
	Header    [0x61a5]byte
	Contacts  [1000]RDTContact `plugger:"Contacts"`
	RxGroups  [250]RDTRxGroup  `plugger:"RxGroups"`
	Zones     [250][64]byte
	ScanLists [250][104]byte
	Padding1  [16]byte
	Channels  [1000][64]byte
	Padding2  [0x11810]byte
}

// Decode reads the contents of the raw RDT plug into a plugger.Plug. The io.EOF
// error is returned when the plug is successfully read but the end of the file
// has been reached. All other errors indicate decoding errors.
func (p RDTPlug) Decode(dst *plugger.Plug) error {
	// Clear out the destination
	dst.Reset()
	// Reflection or code duplication. Take your pick.
	rawVal := reflect.ValueOf(p)
	// Check each field in the raw plug for an appropriate tag. The tag designates
	// the destination field in the Plug to which the contents should be decoded.
	// The `Decode` method is called on each element of the source (if it is a
	// slice) and is copied to the destination.
	for i := 0; i < rawVal.NumField(); i++ {
		srcField := rawVal.Field(i)
		fieldName := rawVal.Type().Field(i).Tag.Get("plugger")

		if fieldName == "" {
			continue
		}

		dstField := reflect.ValueOf(dst).Elem().FieldByName(fieldName)
		if !dstField.IsValid() {
			return DecodeError{fmt.Sprintf("Invalid destination field %s", fieldName)}
		}

		// At this point we have a field in the raw plug which is properly labelled
		// with a destination field. Step through the slice, decoding each element
		// and placing the decoded copy in the destination slice.
		dstElem := reflect.New(dstField.Type().Elem())
		for i := 0; i < srcField.Len(); i++ {
			srcElem := srcField.Index(i)

			dec, ok := srcElem.Interface().(Decoder)
			if !ok {
				return DecodeError{fmt.Sprintf("%s does not implement Decoder", srcElem.Type())}
			}

			decErr := dec.Decode(dstElem.Interface())

			// io.EOF indicates we've consumed all non-blank elements for this raw
			// slice, so skip to the next field.
			if decErr == io.EOF {
				continue
			} else if decErr != nil {
				return decErr
			}

			dstField.Set(reflect.Append(dstField, dstElem.Elem()))
		}
	}

	return nil
}

// RDTDecoder reads RDT formats from an io.Reader into either a raw RDTPlug or
// the more idealised plugger.Plug.
type RDTDecoder struct {
	Reader io.Reader
}

// NewRDTDecoder creates a new decoder for RDT files. Reads successive files
// from the provided io.Reader.
func NewRDTDecoder(r io.Reader) RDTDecoder {
	return RDTDecoder{r}
}

// DecodeRaw reads an RDT file into a raw RDTPlug.
func (dec RDTDecoder) DecodeRaw(dst *RDTPlug) error {
	return binary.Read(dec.Reader, RDTByteOrder, dst)
}

// Decode reads an RDT file into a parsed plugger.Plug.
func (dec RDTDecoder) Decode(dst *plugger.Plug) error {
	// Read a raw plug and then decode that into a plugger.Plug. Capture any
	// io.EOF error (an acceptable state) from the raw decode for possible return.
	raw := RDTPlug{}
	rawErr := dec.DecodeRaw(&raw)
	// io.EOF is okay
	if rawErr != nil && rawErr != io.EOF {
		return rawErr
	}

	err := raw.Decode(dst)
	if err != nil {
		return err
	}
	return rawErr
}

// RDTName is a 16-character UTF-16 label used in MD-380 family radios.
type RDTName [32]byte

// IsBlank returns true if the RDTName has a zero-value without decoding into a
// Go string. Returns false otherwise.
func (n RDTName) IsBlank() bool {
	for _, b := range n {
		if b != 0 {
			return false
		}
	}
	return true
}

// String returns the String value of an encoded RDTName. Used for decoding and
// printing.
func (n RDTName) String() string {
	var buf bytes.Buffer
	r := transform.NewReader(bytes.NewReader(n[:]), RDTNameDecoder)
	buf.ReadFrom(r)
	return buf.String()
}

// RDTContact is a DMR ID, name, and call type used in channels, rx groups, and
// scan lists.
type RDTContact struct {
	ID   [3]byte
	Type byte
	Name RDTName
}

// IsBlank checks the Name and Type of the contact to determine if it is blank.
// Returns true if blank, false otherwise.
func (c RDTContact) IsBlank() bool {
	return bytes.Equal(c.ID[:], []byte{0xff, 0xff, 0xff}) &&
		(c.Type == 0xff || c.Type&0x0f == 0)
}

// Decode reads the RDT contact into a plugger.Contact. Will return a
// DecodeDestError if passed any type except a *plugger.Contact.
func (c RDTContact) Decode(d interface{}) error {
	dst, ok := d.(*plugger.Contact)
	if !ok {
		return DecodeDestError{reflect.TypeOf(d)}
	}

	if c.IsBlank() {
		return io.EOF
	}

	// Append an empty byte
	r := bytes.NewReader(append(c.ID[:], 0))
	// Decode uint24(?!?!) ID
	err := binary.Read(r, RDTByteOrder, &dst.ID)
	if err != nil && err != io.EOF {
		return err
	}

	// Name
	dst.Name = c.Name.String()

	return nil
}

// RDTRxGroup ...
type RDTRxGroup struct {
	Name     RDTName
	Contacts [32]uint16
}

func (g RDTRxGroup) IsBlank() bool {
	return g.Name.IsBlank()
}

func (g RDTRxGroup) Decode(d interface{}) error {
	dst, ok := d.(*plugger.RxGroup)
	if !ok {
		return fmt.Errorf("Not a valid destination: %s", reflect.TypeOf(d))
	}

	if g.IsBlank() {
		return io.EOF
	}

	dst.Name = g.Name.String()
	for _, c := range g.Contacts {
		if c != 0 {
			dst.Contacts = append(dst.Contacts, c)
		}
	}

	return nil
}

// TODO: Metadata
// TODO: Zones
// TODO: ScanLists
// TODO: Channels
