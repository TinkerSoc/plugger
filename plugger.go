package plugger

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"golang.org/x/text/transform"
)

const (
	// ContactCount is the maximum number of contact entries.
	ContactCount = 1000
	// ContactLen is the number of bytes in a contact entry.
	ContactLen = 36
	// ContactOffset is the starting position in the RDT file for contacts.
	ContactOffset = 0x061A5
	// ContactIDBytes is the number of bytes in a contact ID.
	ContactIDBytes = 3
	// ContactIDMax is the highest contact ID which can be encoded.
	ContactIDMax = 16777215

	// LabelLen is the number of bytes permitted in an encoded label.
	LabelLen = 32

	// RxGroupContactCount is the maximum number of contacts in an RX group.
	RxGroupContactCount = 32
	// RxGroupCount is the maximum number of RX Group entries.
	RxGroupCount = 250
	// RxGroupLen is the number of bytes in an RX Group entry.
	RxGroupLen = 96
	// RxGroupOffset is the starting position in the RDT file for RX Groups.
	RxGroupOffset = 0x0EE45

	// ContactBlank is the contact type for a blank entry.
	ContactBlank = 0x00
	// ContactGroup is the contact type for a talkgroup entry.
	ContactGroup = 0x01
	// ContactPrivate is the contact type for a private entry.
	ContactPrivate = 0x02
	// ContactAll is the contact type for an "all" entry, whatever that is
	ContactAll = 0x03
	// ContactTone is the flag for a set incoming call tone.
	ContactTone = 0xE0
	// ContactNoTone is the flag for an unset incoming call tone.
	ContactNoTone = 0xC0
)

// Channel is a DMR channel.
// TODO: (un)marshalling
type Channel struct {
	Name    string
	Type    byte
	Power   byte
	Contact *Contact
	RxGroup *RxGroup
	TxFreq  uint32
	RxFreq  uint32
}

// Contact is a DMR contact with a distinct ID.
type Contact struct {
	ID   uint32
	Type uint
	Tone bool
	Name string
}

// Plug is an abstract representation of a Codeplug file.
// TODO: replace arrays with slices
type Plug struct {
	Contacts  []Contact
	RxGroups  []RxGroup
	Zones     [250]Zone
	ScanLists [250]ScanList
	Channels  [1000]Channel
}

// ScanList is a list of DMR channels used for scanning.
// TODO: (un)marshalling
type ScanList struct {
	Name     string
	Channels [32]*Channel
}

// RxGroup is a list of DMR contacts which a channel will receive.
// TODO: marshalling
type RxGroup struct {
	Name string
	// rawContacts is the list of contact IDs rather than pointers to them.
	// TODO: consider refactoring.
	rawContacts []uint16
	Contacts    []*Contact
}

// Zone is a list of DMR channels which can be selected with the channel knob.
type Zone struct {
	Name     string
	Channels [16]*Channel
}

// NewContact creates a new DMR contact.
func NewContact() Contact {
	c := Contact{}
	return c
}

// MarshalBinary transforms a Contact into the Codeplug binary format. This
// method performs sanity checks during marshalling and will not allow invalid
// data to be encoded. Any io errors will be returned if they occur.
func (c Contact) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	// The contact ID is stored as a 32 bit number in our struct, but is a 24
	// bit number in the file. We strip off the last byte if it's empty, and raise
	// an error otherwise.

	if c.ID > ContactIDMax {
		return nil, fmt.Errorf("Contact ID is too large. Max is %d, %d provided", ContactIDMax, c.ID)
	}
	err := binary.Write(&buf, binary.LittleEndian, &c.ID)

	if err != nil {
		return nil, err
	}

	// Back-step over that final byte we wrote.
	buf.Truncate(buf.Len() - 1)

	// Do bitwise manipulation (upper and lower halves of byte) to calculate the
	// contact type. Upper half describes Tone/NoTone, lower half describes the
	// call type.

	var rawType byte

	if c.Tone {
		rawType = ContactTone
	} else {
		rawType = ContactNoTone
	}

	rawType |= byte(c.Type)

	err = buf.WriteByte(rawType)

	if err != nil {
		return nil, err
	}

	// Here we go through the little-endian UTF-16 encoding used by the file
	// format.
	if len(c.Name) > LabelLen {
		return nil, fmt.Errorf("Contact name longer than allowed. Max is %d bytes, %d provided", LabelLen, len(c.Name))
	}
	t := transform.NewWriter(&buf, LabelEncoding.NewEncoder())
	_, err = t.Write([]byte(c.Name))
	t.Close()

	// Pad the string to fill out all empty space in the contact
	for i, l := 0, buf.Len(); i < ContactLen-l; i++ {
		buf.WriteByte(0x00)
	}

	// Final sanity check for the length of the contact
	if buf.Len() > ContactLen {
		return nil, fmt.Errorf("Contact longer than maximum length. Max is %d, %d generated", ContactLen, buf.Len())
	}

	return buf.Bytes(), nil
}

// UnmarshalBinary transforms a binary representation of a Contact into its
// corresponding struct. Any io errors (except the expected io.EOF) will be
// returned if they occur.
func (c *Contact) UnmarshalBinary(data []byte) error {
	if l := len(data); l != ContactLen {
		return fmt.Errorf("Invalid contact length. Expected %d bytes, got %d.", ContactLen, l)
	}

	buf := bytes.NewReader(data)

	// Read the contact ID.
	// The contact ID is a 24 bit(?!?!) number stored like a uint32, but with the
	// last byte missing.
	// We create a 4 byte buffer, read the real 3 bytes into it, and then do a
	// []byte->uint32 decode using the binary package.
	id := make([]byte, 4)
	n, err := io.LimitReader(buf, ContactIDBytes).Read(id)

	if n != ContactIDBytes {
		return fmt.Errorf("Incorrect contact ID byte count. Expected %d, got %d", ContactIDBytes, n)
	}

	if err == io.EOF {
		err = nil
	}

	if err != nil {
		return err
	}

	err = binary.Read(bytes.NewReader(id), binary.LittleEndian, &c.ID)

	if err != nil {
		return err
	}

	rawType, err := buf.ReadByte()

	if err != nil {
		return err
	}

	switch rawType & 0xF0 {
	case ContactNoTone:
		c.Tone = false
	case ContactTone:
		c.Tone = true
	default:
		return fmt.Errorf("Invalid Contact Type: %x", rawType)
	}

	rawType &= 0x0F

	if ContactBlank <= rawType && rawType <= ContactAll {
		c.Type = uint(rawType)
	} else {
		return fmt.Errorf("Contact type out of range: %x", rawType)
	}

	r := transform.NewReader(io.LimitReader(buf, LabelLen), LabelDecoder)

	rawName := make([]byte, LabelLen/2)
	n, err = r.Read(rawName)

	if n > len(rawName) {
		return fmt.Errorf("Contact Name too long. Expected %d bytes, got %d", len(rawName), n)
	}

	if err == io.EOF {
		err = nil
	}

	if err != nil {
		return err
	}

	c.Name = string(rawName[:n])

	return err
}

// NewPlug creates a new codeplug.
// TODO: Zones, ScanLists, Channels
func NewPlug() Plug {
	p := Plug{
		Contacts: make([]Contact, 0, ContactCount),
		RxGroups: make([]RxGroup, 0, RxGroupCount),
	}

	return p
}

// UnmarshalBinary decodes an RDT format Codeplug and converts all known data
// structures. Pass the entire RDT file to this method. The (expected) io.EOF
// error is caught, all others are passed through.
func (p *Plug) UnmarshalBinary(data []byte) error {
	buf := bytes.NewReader(data)

	//
	// Contacts
	//
	buf.Seek(ContactOffset, 0)
	rawContact := make([]byte, ContactLen)

	for i := 0; i < ContactCount; i++ {
		err := readExactly(rawContact, buf)

		if err != nil {
			return err
		}

		c := NewContact()
		err = c.UnmarshalBinary(rawContact)

		if err != nil {
			return err
		}

		// Ignore contacts which have a "blank" type. They're empty.
		if c.Type != ContactBlank {
			p.Contacts = append(p.Contacts, c)
		}
	}

	//
	// RX Groups
	//
	buf.Seek(RxGroupOffset, 0)
	rawRxGroup := make([]byte, RxGroupLen)

	for i := 0; i < RxGroupCount; i++ {
		err := readExactly(rawRxGroup, buf)

		if err != nil {
			return err
		}

		g := NewRxGroup()

		err = g.UnmarshalBinary(rawRxGroup)

		if err != nil {
			return err
		}

		// Only Rx groups with non-empty names are valid
		if len(g.Name) > 0 {
			for _, c := range g.rawContacts {
				g.Contacts = append(g.Contacts, &p.Contacts[c-1])
			}
			p.RxGroups = append(p.RxGroups, g)
		}
	}

	return nil
}

// NewRxGroup creates a new RxGroup. The Contacts field is initialised with
// the maximum number of DMR contacts supported.
func NewRxGroup() RxGroup {
	g := RxGroup{
		Contacts:    make([]*Contact, 0, RxGroupContactCount),
		rawContacts: make([]uint16, 0, RxGroupContactCount),
	}
	return g
}

// UnmarshalBinary converts a binary representation of an RxGroup. The Contacts
// field cannot be initialised without a codeplug. The Plug.UnmarshalBinary
// method should be used to correctly set the pointers.
func (g *RxGroup) UnmarshalBinary(data []byte) error {

	if l := len(data); l != RxGroupLen {
		return fmt.Errorf("Invalid RX group length. Expected %d bytes, got %d.", RxGroupLen, l)
	}

	buf := bytes.NewReader(data)

	// Read RX Group name
	r := transform.NewReader(io.LimitReader(buf, LabelLen), LabelDecoder)

	rawName := make([]byte, LabelLen/2)
	n, err := r.Read(rawName)

	if n > len(rawName) {
		return fmt.Errorf("RX Group Name too long. Expected %d bytes, got %d", len(rawName), n)
	}

	if err == io.EOF {
		err = nil
	}

	if err != nil {
		return err
	}

	g.Name = string(rawName[:n])

	// Read RX group contact IDs
	var rawContact uint16
	for i := 0; i < RxGroupContactCount; i++ {
		err = binary.Read(buf, binary.LittleEndian, &rawContact)
		if err != nil {
			return err
		}

		if rawContact > 0 {
			g.rawContacts = append(g.rawContacts, rawContact)
		}
	}

	return err
}
