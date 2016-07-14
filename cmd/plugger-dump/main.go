package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/TinkerSoc/plugger"
	"github.com/TinkerSoc/plugger/format"
)

func main() {
	var fileName string
	// Read the filename from the command line
	flag.StringVar(&fileName, "file", "default1.rdt", "RDT file to dump")
	flag.Parse()

	log.Printf("Reading RDT file from %s", fileName)
	// open the RDT file
	f, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	dec := format.NewRDTDecoder(f)
	var dst plugger.Plug

	log.Print("Decoding RDT file")
	start := time.Now()

	err = dec.Decode(&dst)

	log.Printf("Decoded in %s", time.Since(start))

	if err != nil {
		panic(err)
	}

	log.Printf("Plug summary: %d contacts, %d rx groups.", len(dst.Contacts), len(dst.RxGroups))

	for n, c := range dst.Contacts {
		fmt.Printf("Contact %04d: %+v\n", n+1, c)
	}

	for n, g := range dst.RxGroups {
		fmt.Printf("RX Group %03d: %+v\n", n+1, g)
	}

}
