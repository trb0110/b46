package main

import (
	"b46/b46"
	"log"
)

func main() {
	_ = b46.NewEnviroSetup()
	log.Println(b46.Env)

}
