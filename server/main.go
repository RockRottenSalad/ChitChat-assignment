package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"net"
	"sync"

	pb "github.com/augustlh/chitchat/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// source: https://stackoverflow.com/questions/45267125/how-to-generate-unique-random-alphanumeric-tokens-in-golang
func GenerateSecureToken() string {
	b := make([]byte, 128)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

type Client struct {
	username string
	token    string
	stream   pb.ChitChatService_StreamServer
}

type Server struct {
	pb.UnimplementedChitChatServiceServer

	mu        sync.Mutex
	usernames map[string]bool
	clients   map[string]*Client
}

func (s *Server) Connect(ctx context.Context, req *pb.ConnectRequest) (*pb.ConnectResponse, error) {
	username := req.GetUsername()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.usernames[username] {
		err := status.Error(codes.AlreadyExists, "username already in use")
		return nil, err
	}

	client := &Client{
		username: username,
		token:    GenerateSecureToken(),
		stream:   nil,
	}

	s.usernames[username] = true
	s.clients[client.token] = client

	response := &pb.StreamResponse{
		Event: &pb.StreamResponse_Login_{
			Login: &pb.StreamResponse_Login{
				Username: client.username,
			},
		},
	}

	log.Printf("user %v connected", client.username)

	go s.Broadcast(response)

	return &pb.ConnectResponse{Token: client.token}, nil

}

func (s *Server) AuthClient(ctx context.Context) (*Client, error) {
	md, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	tokens := md["authorization"]
	if len(tokens) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing auth token")
	}

	s.mu.Lock()
	token := tokens[0]
	client, exists := s.clients[token]
	if !exists {
		return nil, status.Error(codes.Unauthenticated, "invalid auth token")
	}
	s.mu.Unlock()

	log.Printf("client %v authenticated", client.username)

	return client, nil
}

// Stream is multi-threaded by default
// Whenever a client calls Stream() grpc spawns a new thread through this method
func (s *Server) Stream(stream pb.ChitChatService_StreamServer) error {
	// Auth the user
	client, err := s.AuthClient(stream.Context())
	if err != nil {
		return err
	}

	// Update the stream of the client
	s.mu.Lock()
	client.stream = stream
	s.mu.Unlock()

	for {
		in, err := stream.Recv()
		if err == io.EOF { // If the client called CloseSend()
			s.DisconnectClient(client)
			return nil
		}

		if err != nil {
			return err
		}
		message := in.GetMessage()

		log.Printf("received message %v from user %v", message, client.username)

		response := &pb.StreamResponse{
			Event: &pb.StreamResponse_ChatMessage{
				ChatMessage: &pb.StreamResponse_Message{
					Username: client.username,
					Message:  message,
				},
			},
		}

		go s.Broadcast(response)
	}
}

func (s *Server) DisconnectClient(c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.usernames, c.username)
	delete(s.clients, c.token)

	response := &pb.StreamResponse{
		Event: &pb.StreamResponse_Logout_{
			Logout: &pb.StreamResponse_Logout{
				Username: c.username,
			},
		},
	}

	log.Printf("disconnected client: %v", c.username)
	go s.Broadcast(response)
}

/*func (s *Server) Broadcast(response *pb.StreamResponse) {
	s.mu.Lock()

	var clientToDisconnect sync.Mutex[[]*Client]

	for _, client := range s.clients {
		err := go func(c *Client, dcArray []*Client) error {
			if c.stream == nil {
				return nil
			}
			if err := client.stream.Send(response); err != nil {
				return err
			}

			return nil
		}(client)
	}
	s.mu.Unlock()

}*/

func (s *Server) Broadcast(response *pb.StreamResponse) {
	s.mu.Lock()

	// 1. Create a list to hold clients that fail.
	var clientsToDisconnect []*Client

	// 2. Loop and try to send.
	for _, client := range s.clients {
		if client.stream == nil {
			continue
		}
		if err := client.stream.Send(response); err != nil {
			// 3. DON'T disconnect here. Just add the client to the list.
			log.Printf("Send failed, marking client for removal: %s", client.username)
			clientsToDisconnect = append(clientsToDisconnect, client)
		}
	}

	// 4. IMPORTANT: Unlock the mutex.
	s.mu.Unlock()

	// 5. Now that the lock is free, safely disconnect each client.
	for _, client := range clientsToDisconnect {
		s.DisconnectClient(client) // This is safe. It will get its own lock.
	}
}

func main() {
	ip := "localhost:5001"
	lis, err := net.Listen("tcp", ip)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
		return
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	chitchat := &Server{
		usernames: make(map[string]bool),
		clients:   make(map[string]*Client),
	}

	pb.RegisterChitChatServiceServer(grpcServer, chitchat)
	grpcServer.Serve(lis)
}

//todo fix weird go routine spawning
