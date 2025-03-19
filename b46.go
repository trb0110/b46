package main

import (
	"b46/b46/_sys_init"
	"log"
)

func main() {
	_ = _sys_init.NewEnviroSetup()
	log.Println(_sys_init.Env)

}
