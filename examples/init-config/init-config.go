package main

import (
	"log"
	"neoGo/trading/b46/b46"
)

func main() {
	_ = b46.NewEnviroSetup()
	log.Println(b46.Env)
}
