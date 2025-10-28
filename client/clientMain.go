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

	n := len(os.Args)

	if n == 1 {
		if runtime.GOOS == "windows" {
			runWindows();
		} else {
			runUnix();
		}
	} else {
		switch os.Args[1] {
		case "simple":
			runWindows();
		case "tui":
			runUnix();
		default:
			println("Unknown argument: " + os.Args[1])
			println("Expected one of the following: 'tui', 'simple'")
			println("Defaults to 'tui' on Unix systems and 'simple' on Windows sytems")
		}
	}

}
