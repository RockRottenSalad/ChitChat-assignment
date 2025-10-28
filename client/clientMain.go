package main

import (
	"ChitChat/ui"
	utils "ChitChat/utils"
	"fmt"
	"log"
	"os"
)

type State uint8

const (
	PickUsername State = iota
	PickUsernameRejected
	InChat
	Exit
)

type Application struct {
	cursor      uint
	inputBuffer *utils.FixedArray

	client *Client
	tui    *ui.UI

	messages []ReceivedMessage
	state    State
}

func NewApp() *Application {
	app := new(Application)
	*app = Application{
		cursor:      0,
		inputBuffer: utils.NewFixedArray(128),
		client:      nil,
		tui:         ui.NewUI(),
		state:       PickUsername,
	}

	app.render()

	app.tui.SetCallback(app.handleInput)
	//	app.client.SetCallback(app.handleMessage)

	return app
}

func (app *Application) handleUsernameSubmit() {
	username := app.inputBuffer.String()

	client := NewClient("localhost", "5001", username)

	if client == nil {
		app.state = PickUsernameRejected
	} else {
		app.client = client
		app.client.SetCallback(app.handleMessage)
		app.state = InChat

		app.Log("Client connected to server")
	}
}

func (app *Application) handleSubmit() {

	switch app.state {
	case PickUsername:
		app.handleUsernameSubmit()
	case PickUsernameRejected:
		app.handleUsernameSubmit()
	case InChat:
		app.client.Send(app.inputBuffer.String())
	}

	app.inputBuffer.Reset()
	app.cursor = 0
}

func (app *Application) handleInput(key ui.Key) {
	if key.IsSpecial() {
		switch key.GetSpecial() {
		case ui.Return:
			app.handleSubmit()

		case ui.Esc:
			app.appExit()

		case ui.CtrlC:
			app.appExit()

		case ui.Backspace:
			if app.inputBuffer.Len() > 0 && app.cursor > 0 {
				app.inputBuffer.Delete(app.cursor - 1)
				app.cursor--
			}

		case ui.ArrowLeft:
			if app.cursor > 0 {
				app.cursor--
			}
		case ui.ArrowRight:
			if app.cursor < app.inputBuffer.Len() {
				app.cursor++
			}
		default:
			// TODO: Handle arrow keys
		}
	} else {
		if app.inputBuffer.Len() < app.inputBuffer.Cap() {
			app.inputBuffer.Insert(app.cursor, key.GetLetter())
			app.cursor += 1
		}
	}
	app.render()
}

func (app *Application) handleMessage(msg ReceivedMessage, err error) {
	if err != nil {
		log.Fatalln("TODO: Don't panic")
	} else {
		app.messages = append(app.messages, msg)
	}

	app.Log("Got message: " + fmt.Sprintf("%v", msg))

	if app.state == InChat {
		app.render()
	}
}

func (app *Application) render() {
	switch app.state {
	case PickUsername:
		app.renderStartMenu()
	case PickUsernameRejected:
		app.renderStartMenu()

	case InChat:
		app.renderMessages()
	}

	app.tui.Render()
}

func (app *Application) renderMessages() {
	app.tui.SetCursor(0, 0)
	app.tui.Write("Chit Chat", ui.Red, ui.Default, ui.Underlined)

	for i := range len(app.messages) {
		app.tui.SetCursor(uint(i+1), 2)

		var col ui.Color
		if app.messages[i].author == "You" {
			col = ui.Blue
		} else {
			col = ui.Red
		}

		switch app.messages[i].event {
		case LoginEvent:
			app.tui.Write(fmt.Sprintf("%s @ %d connected to the chat", app.messages[i].author, app.messages[i].lamportTimestamp), ui.Default, ui.Default, ui.Normal)
		case LogoutEvent:
			app.tui.Write(fmt.Sprintf("%s @ %d disconnected from the chat", app.messages[i].author, app.messages[i].lamportTimestamp), ui.Default, ui.Default, ui.Normal)
		case MessageEvent:
			app.tui.Write(fmt.Sprintf("%s @ %d: ", app.messages[i].author, app.messages[i].lamportTimestamp), ui.Default, col, ui.Italic)
			app.tui.Write(app.messages[i].message,
				ui.Default, ui.Default, ui.Normal)
		}
	}

	cursorRow := uint(len(app.messages) + 1)
	app.tui.SetCursor(cursorRow, 0)
	app.tui.Write("> ", ui.Blue, ui.Default, ui.Bold)
	app.tui.Write(app.inputBuffer.String(), ui.Default, ui.Default, ui.Normal)

	app.tui.SetCursor(cursorRow, app.cursor+2)
}

func (app *Application) renderStartMenu() {
	halfHeight := app.tui.GetUIHeight() / 2
	halfWidth := app.tui.GetUIWidth() / 2

	app.tui.SetCursor(halfHeight-1, 0)
	app.tui.WriteCentered("Enter username:", ui.Default, ui.Default, ui.Bold)

	if app.state == PickUsernameRejected {
		app.tui.SetCursor(halfHeight, halfWidth)
		app.tui.WriteCentered("That username is already taken", ui.White, ui.Red, ui.Normal)
	}

	str := app.inputBuffer.String()

	inputStartColumn := halfWidth - uint(len(str)/2)
	app.tui.SetCursor(halfHeight+1, inputStartColumn)
	app.tui.Write(str, ui.Default, ui.Default, ui.Normal)
	app.tui.SetCursor(halfHeight+1, inputStartColumn+app.cursor)
}

func (app *Application) appExit() {
	app.tui.TerminateUI()
	if app.client != nil {
		app.client.Close()
	}
	app.state = Exit
}

func (app *Application) ShouldExit() bool {
	return app.state == Exit
}

func (app *Application) Log(msg string) {
	log.Printf("logical timestamp=\"%v\", component=\"client\", type=\"%v\", username=\"%v\"",
		app.client.clock.Now(), msg, app.client.Username())
}

func main() {
	os.Mkdir("./clientLogs", os.ModePerm)
	logFile := utils.CreateLogFile("./clientLogs/", "clientLog")
	fd, _ := os.Create(logFile)
	log.SetOutput(fd)

	app := NewApp()

	for !app.ShouldExit() {
	}

	app.Log("Client exiting")
}
