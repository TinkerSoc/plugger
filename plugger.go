// Package plugger provides data structures and utilities for manipulating DMR
// codeplugs. At the moment, effort is focused on the MD-380 family of radios
// which have a (simple enough) binary file format. The types defined in this
// package provide high-level abstractions of the data structures used in the
// radio whilst the format package provides platform-specific data structures.
package plugger

type Plug struct {
	Contacts  []Contact
	RxGroups  []RxGroup
	Zones     []Zone
	ScanLists []ScanList
}

func (p *Plug) Reset() {
	p.Contacts = nil
	p.RxGroups = nil
	p.Zones = nil
	p.ScanLists = nil
}

type Contact struct {
	Name string
	ID   uint32
}

type ContactList struct {
	Name     string
	Contacts []*Contact
}

type RxGroup struct {
	Name     string
	Contacts []uint16
}

type Zone struct {
	Name     string
	Contacts []*Contact
}

type ScanList struct {
	ContactList
}

type Channel struct {
	ContactList
}
