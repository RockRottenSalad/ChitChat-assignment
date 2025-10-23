package utils

import (
	"fmt"
	"log"
)

func LogAndPrint(fmtString string, opts ...any) {
	message := fmt.Sprintf(fmtString, opts...)

	fmt.Println(message)
	log.Println(message)
}
