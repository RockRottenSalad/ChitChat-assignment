package utils

import (
	"fmt"
	"log"
	"math/rand"
)

func LogAndPrint(fmtString string, opts ...any) {
	message := fmt.Sprintf(fmtString, opts...)

	fmt.Println(message)
	log.Println(message)
}

func CreateLogFile(dir string, prefix string) string {

	if dir[len(dir)-1] != '/' {
		dir += "/"
	}

	x := rand.Int()
	name := prefix + fmt.Sprintf("%d", x);

	return dir + name
}
