package main

//import (
//	"bufio"
//	"context"
//	"flag"
//	"fmt"
//	"io"
//	"log"
//	"os"
//	"strings"
//	"time"
//
//	"github.com/augustlh/chitchat/logical_clocks"
//	pb "github.com/augustlh/chitchat/proto"
//	"google.golang.org/grpc"
//	"google.golang.org/grpc/credentials/insecure"
//	"google.golang.org/grpc/metadata"
//)
//
//type Client struct {
//	username string
//	token    string
//	client   pb.ChitChatServiceClient
//
//	clock clocks.LamportClock
//}
//
//func Login(client pb.ChitChatServiceClient, username string) (*Client, error) {
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	c := &Client{
//		username: username,
//		token:    "",
//		client:   client,
//		clock:    *clocks.NewLamport(),
//	}
//
//	c.clock.Tick()
//
//	req := &pb.ConnectRequest{
//		Timestamp: c.clock.Now(),
//		Username:  username,
//	}
//
//	res, err := client.Connect(ctx, req)
//	if err != nil {
//		return nil, err
//	}
//
//	c.token = res.GetToken()
//	c.clock.Sync(clocks.From(res.Timestamp))
//
//	return c, nil
//}
//
//func main() {
//	var opts []grpc.DialOption
//	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
//
//	conn, err := grpc.NewClient("localhost:5001", opts...)
//	if err != nil {
//		log.Fatalf("fail to dial: %v", err)
//	}
//	defer conn.Close()
//
//	client := pb.NewChitChatServiceClient(conn)
//
//	username := flag.String("username", "", "Username of client")
//	flag.Parse()
//	c, err := Login(client, *username)
//	if err != nil {
//		log.Fatalf("failed to login with username: %v with reason %v", username, err)
//	}
//
//	md := metadata.New(map[string]string{"authorization": c.token})
//
//	ctx := context.Background()
//	ctxWithMetaData := metadata.NewOutgoingContext(ctx, md)
//
//	stream, err := client.Stream(ctxWithMetaData)
//	if err != nil {
//		log.Fatalf("failed to create stream with server: %v", err)
//	}
//	go func() {
//		for {
//			in, err := stream.Recv()
//			if err == io.EOF {
//				log.Printf("Server closed stream")
//				return
//			}
//
//			if err != nil {
//				log.Fatalf("error reading from stream: %v", err)
//				return
//			}
//			eventTimestamp := c.clock.Sync(clocks.From(in.Timestamp))
//
//			switch event := in.Event.(type) {
//			case *pb.StreamResponse_ChatMessage:
//				if event.ChatMessage.Username == *username {
//					fmt.Printf("\r[%v] [you]: %s\n> ", eventTimestamp, event.ChatMessage.Message)
//				} else {
//					fmt.Printf("\r[%v] [%s]: %s\n> ", eventTimestamp, event.ChatMessage.Username, event.ChatMessage.Message)
//				}
//
//			case *pb.StreamResponse_LoginEvent:
//				fmt.Printf("\r*** [%v] %s joined the chat ***\n> ", eventTimestamp, event.LoginEvent.Username)
//
//			case *pb.StreamResponse_LogoutEvent:
//				fmt.Printf("\r*** [%v] %s left the chat ***\n> ", eventTimestamp, event.LogoutEvent.Username)
//
//			default:
//				fmt.Printf("\r[%v] Unknown event: %T\n> ", eventTimestamp, event)
//			}
//		}
//	}()
//
//	reader := bufio.NewReader(os.Stdin)
//	fmt.Println("Connected! Type your message and press Enter.")
//	fmt.Print("> ")
//
//	for {
//		text, _ := reader.ReadString('\n')
//		text = strings.TrimSpace(text)
//		if text == "" {
//			continue
//		}
//		if strings.ToLower(text) == "/quit" {
//			fmt.Println("Exiting chat...")
//			break
//		}
//
//		// Truncate to 128 chars
//		if len(text) > 128 {
//			text = text[:125] + "..."
//		}
//
//		c.clock.Tick()
//
//		err := stream.Send(&pb.StreamRequest{
//			Timestamp: c.clock.Now(),
//			Message:   text,
//		})
//		if err != nil {
//			log.Fatalf("Error sending message: %v", err)
//		}
//
//		fmt.Print("> ")
//	}
//
//	stream.CloseSend()
//
//}
