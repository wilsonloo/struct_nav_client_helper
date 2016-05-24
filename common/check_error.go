package common

import (
	"fmt"
	"log"
	"os"
)

func CheckErrorOrExit(err error, args ...interface{}) {
	if err != nil {
		log.Fatalln("Error: ", err.Error())

		if args != nil {
			fmt.Println(len(args))
		}
		os.Exit(1)
	}
}
