package logging

import (
	"log"
)

func PrintErrorToLog(errorDescription string, errorString interface{}) {
	log.Println(errorDescription, errorString.(string))
}
