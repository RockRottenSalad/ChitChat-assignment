package main;

import (
	"log"
	"fmt"
	"strings"
	"ChitChat/ui"
)

type State uint8
const (
	PickUsername State = iota
	InChat
	Exit
)
type Application struct {
	inputBuffer *strings.Builder
	client *Client
	tui *ui.UI

	messages []ReceivedMessage
	state State
}

func NewApp() *Application {
	app := new(Application) 
	*app = Application {
		inputBuffer: &strings.Builder{},
		client: NewClient("127.0.0.1", "8080"),
		tui: ui.NewUI(),
		state: InChat,
	}

	app.renderMessages()

	app.tui.SetCallback(app.handleInput)
	app.client.SetCallback(app.handleMessage)

	return app
}

func (app *Application) handleInput(key ui.Key) {
	if key.IsSpecial()  {
		switch key.GetSpecial() {
		case ui.Return:
		app.client.Send(app.inputBuffer.String())
		app.inputBuffer.Reset()
		case ui.Esc:
		app.appExit()
		default:
		// TODO: Handle arrow keys
		}
	} else {
		app.inputBuffer.WriteRune(key.GetLetter())
	}
	app.render()
}

func (app *Application) handleMessage(msg ReceivedMessage, err error) {
	if err != nil {
		log.Fatalln("TODO: Don't panic")
	}else {
		app.messages = append(app.messages, msg)
	}

	app.renderMessages()
}

func (app *Application) render() {
	switch app.state {
	case PickUsername:
	app.renderStartMenu()

	case InChat:
	app.renderMessages()
	}
}

func (app *Application) renderMessages() {
	app.tui.SetCursor(0, 0)
	app.tui.Write("Chit Chat", ui.Red, ui.Default, ui.Underlined)
	id := app.client.Id()

	for i := range len(app.messages) {
		app.tui.SetCursor(uint(i + 1), 2)

		var col ui.Color
		if app.messages[i].author == id {
			col = ui.Blue
		} else {
			col = ui.Red
		}

		// Replace author with actual username in future,
		// currently auhtor is included in the message
		app.tui.Write(fmt.Sprintf("Client %d @ %d:",
			app.messages[i].author,
			app.messages[i].lamportTimestamp),
			ui.Default, col, ui.Italic)

		app.tui.Write(app.messages[i].message,
			ui.Default, ui.Default, ui.Normal)
	}

	app.tui.SetCursor(uint(len(app.messages) + 1), 0)
	app.tui.Write("> ", ui.Blue, ui.Default, ui.Bold)
	app.tui.Write(app.inputBuffer.String(), ui.Default, ui.Default, ui.Normal)

	app.tui.Render()
}

func (app *Application) renderStartMenu() {
	halfHeight := app.tui.GetUIHeight() / 2
	halfWidth := app.tui.GetUIWidth() / 2

	app.tui.SetCursor(halfHeight - 1, 0)
	app.tui.WriteCentered("Enter username:", ui.Default, ui.Default, ui.Bold)
	app.tui.SetCursor(halfHeight + 1, halfWidth)
	app.tui.Render()
}

func (app *Application) appExit() {
	app.state = Exit
	app.tui.TerminateUI()
	app.client.Close()
}

func (app *Application) ShouldExit() bool {
	return app.state == Exit
}


func main() {
	app := NewApp()

	for !app.ShouldExit() {}

	log.Printf("Client: Exiting")
}

