package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/TinkerSoc/plugger"
)

func main() {
	var fileName string
	// Read the filename from the command line
	flag.StringVar(&fileName, "file", "default1.rdt", "RDT file to dump")

	flag.Parse()

	// open the RDT file
	f, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Create a new Plug struct and read RDT file contents to memory.
	plug := plugger.NewPlug()
	b, err := ioutil.ReadAll(f)

	if err != nil {
		panic(err)
	}

	// Unmarshal the Plug
	err = plug.UnmarshalBinary(b)

	// io.EOF is okay.
	if err != nil && err != io.EOF {
		panic(err)
	}

	for i, c := range plug.Contacts {
		fmt.Printf("Contact #%04d: %+v\n", i+1, c)
	}

	fmt.Println()

	for i, g := range plug.RxGroups {
		fmt.Printf("RX Group %d: %+v\n", i+1, g)
	}

	fmt.Println()
	fmt.Printf("Number of contacts: %d\n", len(plug.Contacts))
	fmt.Printf("Number of rx groups: %d\n", len(plug.RxGroups))
}
