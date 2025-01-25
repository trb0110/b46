package b46

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Enviro struct {
	RPC_MAIN string
	RPC_DEV  string
	WSS_MAIN string
	WSS_DEV  string
	PK_MAIN  string
	PK_DEV   string
}

var Env *Enviro

func NewEnviroSetup() *Enviro {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	Env = &Enviro{
		RPC_MAIN: os.Getenv("RPC_MAIN"),
		WSS_MAIN: os.Getenv("WSS_MAIN"),
		PK_MAIN:  os.Getenv("PK_MAIN"),
	}
	return Env
}
