package b46

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Enviro struct {
	RPC         string
	WSS         string
	PK          string
	DEVELOPMENT string
}

var Env *Enviro

func NewEnviroSetup() *Enviro {
	envFile := ".env"
	err := godotenv.Load(envFile)
	if err != nil {
		log.Println("Error loading .env file: " + err.Error())
	}

	env := os.Getenv("DEVELOPMENT")
	if env == "TRUE" {
		envFile = ".env.dev"
		errLoadDev := godotenv.Overload(envFile)
		if errLoadDev != nil {
			log.Println("Error loading .env file: " + errLoadDev.Error())
		}
	}
	Env = &Enviro{
		RPC:         os.Getenv("RPC"),
		WSS:         os.Getenv("WSS"),
		PK:          os.Getenv("PK"),
		DEVELOPMENT: os.Getenv("DEVELOPMENT"),
	}
	return Env
}
