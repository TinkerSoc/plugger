package plugger

import (
	"reflect"
	"testing"
)

type testCaseContact struct {
	contact Contact
	raw     []byte
}

// testContacts are test cases for the (un)marshalling of contacts.
var testContacts = []testCaseContact{
	{
		contact: Contact{Name: "Valid 1", Type: ContactGroup, ID: 1, Tone: false},
		raw: []byte{0x01, 0x00, 0x00, 0xC1, 0x56, 0x00, 0x61, 0x00, 0x6C, 0x00, 0x69, 0x00,
			0x64, 0x00, 0x20, 0x00, 0x31, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	},
	{
		contact: Contact{Name: "Valid 2", Type: ContactGroup, ID: 2, Tone: true},
		raw: []byte{0x02, 0x00, 0x00, 0xE1, 0x56, 0x00, 0x61, 0x00, 0x6C, 0x00, 0x69, 0x00,
			0x64, 0x00, 0x20, 0x00, 0x32, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	},
	{
		contact: Contact{Name: "Valid 3", Type: ContactPrivate, ID: 3, Tone: false},
		raw: []byte{0x03, 0x00, 0x00, 0xC2, 0x56, 0x00, 0x61, 0x00, 0x6C, 0x00, 0x69, 0x00,
			0x64, 0x00, 0x20, 0x00, 0x33, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	},
	{
		contact: Contact{Name: "Valid 4", Type: ContactPrivate, ID: 4, Tone: true},
		raw: []byte{0x04, 0x00, 0x00, 0xE2, 0x56, 0x00, 0x61, 0x00, 0x6C, 0x00, 0x69, 0x00,
			0x64, 0x00, 0x20, 0x00, 0x34, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	},
	{
		contact: Contact{Name: "Valid 5", Type: ContactAll, ID: 16777215, Tone: false},
		raw: []byte{0xFF, 0xFF, 0xFF, 0xC3, 0x56, 0x00, 0x61, 0x00, 0x6C, 0x00, 0x69, 0x00,
			0x64, 0x00, 0x20, 0x00, 0x35, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	},
}

// testDeepEqual takes a format string, expected, and actual values. If the
// values do not match (using reflect.DeepEqual), the values are printed
// together and a testing error state is set.
func testDeepEqual(t *testing.T, f string, exp, actual interface{}) bool {
	res := reflect.DeepEqual(exp, actual)
	if !res {
		t.Logf("Expected value: "+f+"\n", exp)
		t.Logf("Actual   value: "+f+"\n", actual)

		t.Errorf("Expected and actual values do not match")
	}
	return res
}

// TestContactBinaryMarshal checks that the defined test cases marshal properly
// from structs to binary.
func TestContactBinaryMarshal(t *testing.T) {
	for i, test := range testContacts {
		t.Logf("Marshalling contact %d...\n", i)
		res, err := test.contact.MarshalBinary()

		if err != nil {
			t.Error(err)
		}

		testDeepEqual(t, "%x", res, test.raw)
	}
}

// TestContactBinaryUnmarshal checks that the defined test cases unmarshal
// properly from binary to structs.
func TestContactBinaryUnmarshal(t *testing.T) {
	for i, test := range testContacts {
		t.Logf("Unmarshalling contact %d...\n", i)
		res := NewContact()
		err := res.UnmarshalBinary(test.raw)

		if err != nil {
			t.Error(err)
		}

		testDeepEqual(t, "%+v", res, test.contact)
	}
}