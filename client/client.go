// client.go
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	chitchatpb "itu_chitchat/grpc"
	"itu_chitchat/lamport"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func clearLine() { fmt.Fprint(os.Stdout, "\r\033[K") }

func main() {
	fmt.Print("Enter your name: ")
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return
	}
	name := strings.TrimSpace(sc.Text())
	if name == "" {
		return
	}

	conn, err := grpc.NewClient(
		"dns:///localhost:5050",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := chitchatpb.NewChatClient(conn)
	stream, err := client.ChatStream(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	var clk lamport.Clock
	clientID := name

	go func() {
		for {
			m, err := stream.Recv()
			if err != nil {
				log.Println("stream closed:", err)
				return
			}
			clk.Observe(m.GetLamport())
			clearLine()
			fmt.Printf("[L=%d] %s: %s\n", clk.Read(), m.GetUser(), m.GetText())
			fmt.Print("> ")
		}
	}()

	fmt.Print("> ")
	for sc.Scan() {
		text := strings.TrimSpace(sc.Text())
		if text == "" {
			fmt.Print("> ")
			continue
		}
		if text == "/quit" {
			_ = stream.CloseSend()
			break
		}

		ts := clk.Tick()
		msg := &chitchatpb.ChatMessage{
			User:     name,
			Text:     text,
			Lamport:  ts,
			ClientId: clientID,
		}
		if err := stream.Send(msg); err != nil {
			log.Println("send error:", err)
			break
		}

		clearLine()
		fmt.Print("> ")
	}

	if err := sc.Err(); err != nil {
		log.Println("input error:", err)
	}
}
