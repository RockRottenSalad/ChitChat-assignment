package main;

import (
	"log"
	"os"
	"bufio"
)

func reader(client *Client, ch chan bool) {
	reader := bufio.NewReader(os.Stdin)

	for {
		bytes, _, _ := reader.ReadLine()
		text := string(bytes)

		if text == "exit" {
			log.Println("Client: Exiting...")
			client.Close()
			ch <- true
			return
		}

		err := client.Send(text)
		log.Printf("Client: Sending - %s\n", text)
		if err != nil {
			ch <- true
			log.Printf("Client: Got error from server on send - %s\n", err.Error())
		}

	}
}


func main() {
	client := NewClient("127.0.0.1", "8080")

	ch := make(chan bool)

	go reader(&client, ch)
	out:
	for {

		select {
			case <- ch:
			break out
			default:
		}

		rep, err := client.Recv()

		if err != nil {
			log.Fatalf("Client: Got error on recv - %s\n", err.Error())
		}

		log.Println(rep)
	}

	log.Printf("Client: Exiting")
}

