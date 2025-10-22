package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"net"
	"sync"

	"github.com/augustlh/chitchat/logical_clocks"
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

	// The clock implementation is thread safe by default
	clock clocks.LamportClock

	mu        sync.Mutex
	usernames map[string]bool
	clients   map[string]*Client
}

func (s *Server) Connect(ctx context.Context, req *pb.ConnectRequest) (*pb.ConnectResponse, error) {
	clientClock := clocks.From(req.Timestamp)
	eventTimestamp := s.clock.Sync(clientClock)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.usernames[req.Username] {
		err := status.Error(codes.AlreadyExists, "username already in use")
		return nil, err
	}

	client := &Client{
		username: req.Username,
		token:    GenerateSecureToken(),
		stream:   nil,
	}

	s.usernames[client.username] = true
	s.clients[client.token] = client

	s.clock.Tick()
	response := &pb.StreamResponse{
		Timestamp: s.clock.Now(),
		Event: &pb.StreamResponse_LoginEvent{
			LoginEvent: &pb.StreamResponse_Login{
				Username: client.username,
			},
		},
	}

	log.Printf("user %v connected at time %v", client.username, eventTimestamp)

	go s.Broadcast(response)

	return &pb.ConnectResponse{Timestamp: eventTimestamp, Token: client.token}, nil

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
		eventTimestamp := s.clock.Sync(clocks.From(in.Timestamp))

		message := in.GetMessage()
		log.Printf("received message %v from user %v at %v", message, client.username, eventTimestamp)

		s.clock.Tick()
		response := &pb.StreamResponse{
			Timestamp: s.clock.Now(),
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

	s.clock.Tick()
	response := &pb.StreamResponse{
		Timestamp: s.clock.Now(),
		Event: &pb.StreamResponse_LogoutEvent{
			LogoutEvent: &pb.StreamResponse_Logout{
				Username: c.username,
			},
		},
	}

	log.Printf("disconnected client: %v", c.username)
	go s.Broadcast(response)
}

func (s *Server) Broadcast(response *pb.StreamResponse) {
	s.mu.Lock()
	var clientsToDisconnect []*Client

	for _, client := range s.clients {
		if client.stream == nil {
			continue
		}
		if err := client.stream.Send(response); err != nil {
			log.Printf("Send failed, marking client for removal: %s", client.username)
			clientsToDisconnect = append(clientsToDisconnect, client)
		}
		//todo: do we anna tick here s.clock.Tick()
	}

	s.mu.Unlock()
	for _, client := range clientsToDisconnect {
		s.DisconnectClient(client)
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
		clock:     *clocks.NewLamport(),
	}

	pb.RegisterChitChatServiceServer(grpcServer, chitchat)
	grpcServer.Serve(lis)
}

//todo use channels
