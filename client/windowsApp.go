package main

import (
	"log"
	"os"
	"fmt"
	"bufio"
	"strings"
)

func inputReader(ch chan string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		str, _ := reader.ReadString('\n')
		ch <- strings.TrimSpace(str)
	}
}

func msgReceiver(client *Client, ch chan ReceivedMessage) {
	for {
		msg, err := client.recv()
		ch <- msg

		// err is also included in msg, so the client is well aware
		if err != nil { return }
	}
}

func handleMessage(msg *ReceivedMessage) bool {
	switch msg.event {
	case MessageEvent:
		fmt.Printf("%s @ %d: %s\n", msg.author, msg.lamportTimestamp, msg.message)
	case LoginEvent:
		fmt.Printf("%s @ %d: connected to the chat\n", msg.author, msg.lamportTimestamp)
	case LogoutEvent:
		fmt.Printf("%s @ %d: disconnected from the chat\n", msg.author, msg.lamportTimestamp)
	case ErrEvent:
		println("Got error")
		return false
	} 

	return true
}

func Log(message string, client *Client) {
	log.Printf("logical timestamp=\"%v\", component=\"client\", type=\"%v\", username=\"%v\"",
		client.clock.Now(), message, client.Username())
}

func Windows() {
	var client *Client

	inputCh := make(chan string)
	msgCh := make(chan ReceivedMessage)

	go inputReader(inputCh)

	for {
		println("Pick username:");
		enableCallback := false
		client = NewClient("localhost", "5001", <-inputCh, enableCallback)

		if client == nil {
			println("That username is already in use")
		} else {
			break
		}
	}

	go msgReceiver(client, msgCh)
	println("You are now connected to the esrver")
	Log("Connected to server", client)

	var running = true
	for running {
		select {
		case input := <- inputCh:
		if client.Send(input) != nil {
			println("Failed to send message")
			running = false 
			Log("Failed to send message: " + input, client)
		}
			Log("Sent message: " + input, client)
		case msg := <- msgCh:
			running = handleMessage(&msg)
			Log("Got message: " + msg.message, client)
		}
	}

	Log("Client closing", client)
}
