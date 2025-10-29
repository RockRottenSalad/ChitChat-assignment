package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"net"
	"os"
	"sync"

	pb "ChitChat/grpc"
	clocks "ChitChat/logical_clocks"
	"ChitChat/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
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
	send     chan *pb.StreamResponse
}

type Server struct {
	pb.UnimplementedChitChatServiceServer

	clock clocks.LamportClock

	mu        sync.Mutex
	usernames map[string]bool
	clients   map[string]*Client
}

func (s *Server) Connect(ctx context.Context, req *pb.ConnectRequest) (*pb.ConnectResponse, error) {
	clientClock := clocks.From(req.Timestamp)
	eventTimestamp := s.clock.Sync(clientClock)

	peer, _ := peer.FromContext(ctx)

	utils.LogAndPrint("logical timestamp=\"%v\", component=\"server\", type=\"login request\", ip=\"%s\", username=\"%s\"", eventTimestamp, peer.Addr.String(), req.Username)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.usernames[req.Username] {
		utils.LogAndPrint("logical timestamp=\"%v\", component=\"server\", type=\"refused login request\", ip=\"%s\", username=\"%s\", reason=\"username already exists\"", eventTimestamp, peer.Addr.String(), req.Username)
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

	utils.LogAndPrint("logical timestamp=\"%v\", component=\"server\", type=\"connect sucess\", ip=\"%s\", username=\"%s\"", eventTimestamp, peer.Addr.String(), req.Username)

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

	utils.LogAndPrint("logical timestamp=\"%v\", component=\"server\", type=\"auth sucess\", username=\"%s\"", s.clock.Now(), client.username)
	return client, nil
}

func (c *Client) ClientBroadcasterHandler(ctx context.Context, errorChan chan error) {
	for {
		select {
		case <-ctx.Done():
			errorChan <- ctx.Err()
			return
		case msg := <-c.send:
			if err := c.stream.Send(msg); err != nil {
				errorChan <- err
				return
			}
		}
	}
}

// Stream is multi-threaded by default
// Whenever a client calls Stream() grpc spawns a new thread through this method
func (s *Server) Stream(stream pb.ChitChatService_StreamServer) error {
	// Auth the user
	peer, _ := peer.FromContext(stream.Context())
	utils.LogAndPrint("logical timestamp=\"%v\", component=\"server\", type=\"stream request\", ip=\"%v\"", s.clock.Now(), peer.Addr.String())
	client, err := s.AuthClient(stream.Context())
	if err != nil {
		return err
	}

	// Update the stream of the client
	// Create channel for communication
	client.stream = stream
	client.send = make(chan *pb.StreamResponse, 32)

	// Spawns goroutine to handle broadcasting to the client
	errorChan := make(chan error, 1)
	go client.ClientBroadcasterHandler(stream.Context(), errorChan)

	for {
		select {
		case <-errorChan:
			s.DisconnectClient(client)
			return nil
		default:
		}

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

		if len(message) > 128 {
			utils.LogAndPrint("logical timestamp=\"%v\", component=\"server\", type=\"message too long\", username=\"%v\"", eventTimestamp, client.username)
			continue
		}

		utils.LogAndPrint("logical timestamp=\"%v\", component=\"server\", type=\"received message\", username=\"%v\", message=\"%v\"", eventTimestamp, client.username, message)
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

	t := s.clock.Now()

	s.clock.Tick()
	response := &pb.StreamResponse{
		Timestamp: t,
		Event: &pb.StreamResponse_LogoutEvent{
			LogoutEvent: &pb.StreamResponse_Logout{
				Username: c.username,
			},
		},
	}

	utils.LogAndPrint("logical timestamp=\"%v\", component=\"server\", type=\"disconnected client\", username=\"%v\"", t, c.username)
	go s.Broadcast(response)
}

func (s *Server) Broadcast(response *pb.StreamResponse) {
	utils.LogAndPrint("logical timestamp=\"%v\", component=\"server\", type=\"broadcast\", message=\"%v\"", response.Timestamp, response)
	s.mu.Lock()
	for _, client := range s.clients {
		select {
		case client.send <- response:
		default: // If a client is slow (their send channel is full) we simply drop the messages
			continue
		}
	}
	s.mu.Unlock()
	s.clock.Tick()
}

func main() {
	f, err := os.OpenFile("serverlogfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

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
	utils.LogAndPrint("server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	utils.LogAndPrint("Stopping server...")
}
