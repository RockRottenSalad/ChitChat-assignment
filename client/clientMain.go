package main;

import (
	"log"
	"fmt"
	"ChitChat/ui"
	utils "ChitChat/utils"
)

type State uint8
const (
	PickUsername State = iota
	InChat
	Exit
)
type Application struct {
	cursor uint
	inputBuffer *utils.FixedArray

	client *Client
	tui *ui.UI

	messages []ReceivedMessage
	state State
}

func NewApp() *Application {
	app := new(Application) 
	*app = Application {
		cursor: 0,
		inputBuffer: utils.NewFixedArray(64),
		client: NewClient("127.0.0.1", "8080"),
		tui: ui.NewUI(),
		state: PickUsername,
	}

	app.render()

	app.tui.SetCallback(app.handleInput)
	app.client.SetCallback(app.handleMessage)

	return app
}

func (app *Application) handleSubmit() {
	switch app.state {
	case PickUsername:
		app.state = InChat
	case InChat:
		app.client.Send(app.inputBuffer.String())
	}

	app.inputBuffer.Reset()
	app.cursor = 0
}

func (app *Application) handleInput(key ui.Key) {
	if key.IsSpecial()  {
		switch key.GetSpecial() {
		case ui.Return:
			app.handleSubmit()

		case ui.Esc | ui.CtrlC:
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
		app.inputBuffer.Insert(app.cursor, key.GetLetter())
		app.cursor += 1
	}

	app.render()
}

func (app *Application) handleMessage(msg ReceivedMessage, err error) {
	if err != nil {
		log.Fatalln("TODO: Don't panic")
	}else {
		app.messages = append(app.messages, msg)
	}

	if app.state == InChat {
		app.render()
	}
}

func (app *Application) render() {
	switch app.state {
	case PickUsername:
	app.renderStartMenu()

	case InChat:
	app.renderMessages()
	}

	app.tui.Render()
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

	cursorRow := uint(len(app.messages) + 1)
	app.tui.SetCursor(cursorRow, 0)
	app.tui.Write("> ", ui.Blue, ui.Default, ui.Bold)
	app.tui.Write(app.inputBuffer.String(), ui.Default, ui.Default, ui.Normal)

	app.tui.SetCursor(cursorRow, app.cursor + 2)
}

func (app *Application) renderStartMenu() {
	halfHeight := app.tui.GetUIHeight() / 2
	halfWidth := app.tui.GetUIWidth() / 2

	app.tui.SetCursor(halfHeight - 1, 0)
	app.tui.WriteCentered("Enter username:", ui.Default, ui.Default, ui.Bold)
	app.tui.SetCursor(halfHeight + 1, halfWidth)
	app.tui.Write(app.inputBuffer.String(), ui.Default, ui.Default, ui.Normal)
	app.tui.SetCursor(halfHeight + 1, halfWidth + app.cursor)
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
	log.Printf("Client did shit")
	app := NewApp()


	for !app.ShouldExit() {}

	log.Printf("Client: Exiting")
}

