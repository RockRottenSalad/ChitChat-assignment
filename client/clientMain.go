package main

import (
	utils "ChitChat/utils"
	"log"
	"os"
	"runtime"
)

func runUnix() {
	app := NewApp()

	for !app.ShouldExit() { }

	app.Log("Client exiting")
}


func runWindows() {
	Windows();
}

func main() {
	os.Mkdir("./clientLogs", os.ModePerm)
	logFile := utils.CreateLogFile("./clientLogs/", "clientLog")
	fd, _ := os.Create(logFile)
	log.SetOutput(fd)

	if runtime.GOOS == "windows" {
		runWindows();
	} else {
		runUnix();
	}

}
