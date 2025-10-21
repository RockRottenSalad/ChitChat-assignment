package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	pb "github.com/augustlh/chitchat/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func Login(client pb.ChitChatServiceClient, username string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &pb.ConnectRequest{
		Username: username,
	}

	res, err := client.Connect(ctx, req)
	if err != nil {
		return "", err
	}

	return res.GetToken(), nil
}

func main() {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient("localhost:5001", opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewChitChatServiceClient(conn)

	username := flag.String("username", "", "Username of client")
	flag.Parse()
	token, err := Login(client, *username)
	if err != nil {
		log.Fatalf("failed to login with username: %v with reason %v", username, err)
	}

	md := metadata.New(map[string]string{"authorization": token})

	ctx := context.Background()
	ctxWithMetaData := metadata.NewOutgoingContext(ctx, md)

	stream, err := client.Stream(ctxWithMetaData)
	if err != nil {
		log.Fatalf("failed to create stream with server: %v", err)
	}
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				log.Printf("Server closed stream")
				return
			}

			if err != nil {
				log.Fatalf("error reading from stream: %v", err)
				return
			}

			switch event := in.Event.(type) {
			case *pb.StreamResponse_ChatMessage:
				if event.ChatMessage.Username == *username {
					fmt.Printf("\r[you]: %s\n> ", event.ChatMessage.Message)
				} else {
					fmt.Printf("\r[%s]: %s\n> ", event.ChatMessage.Username, event.ChatMessage.Message)
				}

			case *pb.StreamResponse_Login_:
				fmt.Printf("\r*** %s joined the chat ***\n> ", event.Login.Username)

			case *pb.StreamResponse_Logout_:
				fmt.Printf("\r*** %s left the chat ***\n> ", event.Logout.Username)

			default:
				fmt.Printf("\r[Unknown event: %T]\n> ", event)
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Connected! Type your message and press Enter.")
	fmt.Print("> ")

	for {
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if strings.ToLower(text) == "/quit" {
			fmt.Println("Exiting chat...")
			break
		}

		// Truncate to 128 chars
		if len(text) > 128 {
			text = text[:125] + "..."
		}

		err := stream.Send(&pb.StreamRequest{
			Message: text,
		})
		if err != nil {
			log.Fatalf("Error sending message: %v", err)
		}

		fmt.Print("> ")
	}

	stream.CloseSend()

}
