package main;

import (
	"log"
	"os"
	"fmt"
	"bufio"
	"ChitChat/ui"
)

func stdinReader(ch chan string) {
	reader := bufio.NewReader(os.Stdin)

	for {
		bytes, _, _ := reader.ReadLine()
		ch <- string(bytes)
	}
}

func streamReader(client *Client, reps chan ReceivedMessage, errs chan error) {
	for {
		rep, err := client.Recv()

		if err != nil {
			errs <- err
			return
		}

		reps <- rep
	}
}

func renderMessages(u *ui.UI, id uint32, messages []ReceivedMessage) {
	u.SetCursor(0, 0)
	u.Write("Chit Chat", ui.Red, ui.Default, ui.Underlined)

	for i := range len(messages) {
		u.SetCursor(uint(i + 1), 2)

		var col ui.Color
		if messages[i].author == id {
			col = ui.Blue
		} else {
			col = ui.Red
		}

		// Replace author with actual username in future,
		// currently auhtor is included in the message
		u.Write(fmt.Sprintf("Client %d @ %d:", messages[i].author, messages[i].lamportTimestamp), ui.Default, col, ui.Italic)
		u.Write(messages[i].message, ui.Default, ui.Default, ui.Normal)
	}

	u.SetCursor(uint(len(messages) + 1), 0)
	u.Write("> ", ui.Blue, ui.Default, ui.Bold)

	u.Render()
}

func main() {
	ui := ui.NewUI()

	client := NewClient("127.0.0.1", "8080")

	inputs := make(chan string)
	reps := make(chan ReceivedMessage)
	errs := make(chan error)

	defer close(inputs)
	defer close(reps)
	defer close(errs)
	defer client.Close()

	var messages []ReceivedMessage
	renderMessages(&ui, client.id, messages)

	go streamReader(&client, reps, errs)
	go stdinReader(inputs)
	out:
	for {

		select {
		case input := <- inputs:
			err := client.Send(input)
			if err != nil {
				log.Printf("Client got error on send - %s\n", err.Error())
				break out
			}
		case rep := <- reps:
			messages = append(messages, rep)
			renderMessages(&ui, client.id, messages)
		case err := <- errs:
			log.Printf("Cilent: Got error on recv - %s\n", err.Error())
			break out
		}

	}

	log.Printf("Client: Exiting")
}

