package main

import (
	"b46/b46/_sys_init"
	"b46/b46/logging"
	"b46/b46/strategies"
	"github.com/coder/websocket"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	_ = _sys_init.NewEnviroSetup()
	//log.Println(_sys_init.Env)

	errLogSession := logging.InitLogSession()
	if errLogSession != nil {
		logging.PrintErrorToLog("Error creating logging session:		", errLogSession.Error())
	}

	kamikaze := strategies.Kamikaze{}
	kamikaze.InitializeKamikaze()

	go kamikaze.Start()

	defer kamikaze.Websocket.Close(websocket.StatusInternalError, "Connection closed")
	defer kamikaze.WssClient.Close()

	// Listen for interrupt signals to gracefully shut down.
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTRAP)
	<-quit
	log.Println("Shutting down B46...")
	logging.CloseAllLoggers()
	log.Println("Closed All Loggers...")

}
