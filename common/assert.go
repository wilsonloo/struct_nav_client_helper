package common

import (
	"log"
	"os"
)

func Assert(flag bool, args ...interface{}) {
	if !flag {
		log.Fatalln("Assert: ", args)
		os.Exit(1)
	}
}
