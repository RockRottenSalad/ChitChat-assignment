package utils

import "log"

func Assert(cond bool, message string) {
	if !cond {
		log.Fatalln(message)
	}
}

