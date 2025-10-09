package main;

import (
	proto "ChitChat/grpc"
	"net"
	"log"
	"io"
	"context"
	"sync"
	"fmt"
	"google.golang.org/grpc"
)

type ClientConnection struct {
	stream proto.MessageService_ConnectServer
}

type Server struct {
	proto.UnimplementedMessageServiceServer
	clientCount int
	clients []*ClientConnection
	mu sync.Mutex
};

func NewServer() Server {
	return Server{clientCount: 0}
}

func (s *Server) AddClient(client *ClientConnection) int {
	s.mu.Lock()

	id := len(s.clients)
	s.clients = append(s.clients, client)
	s.clientCount += 1

	s.mu.Unlock()

	return id
}

func (s *Server) RemoveClient(index int) {
	s.mu.Lock()

	n := len(s.clients)
	s.clients[index], s.clients[n-1] = s.clients[n-1], s.clients[index]
	s.clients = s.clients[:n-1]
	s.clientCount -= 1

	s.mu.Unlock()
}

func (s *Server) Replicate(id int, message *proto.Message) {
	for i := range s.clientCount {
		if i == id { continue; }
		err := s.clients[i].stream.Send(message)
		if err != nil { log.Println("Server: Client died - Should inform handler in the future")  }
	}
}

func (s *Server) StartServer() {
	log.Println("Server: Starting...");

	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Server: Failed to bind to port")
	}

	proto.RegisterMessageServiceServer(grpcServer, s)

	log.Println("Server: Serving message service on port 8080");
	err = grpcServer.Serve(listener)

	if err != nil {
		log.Fatalf("Server: Failed to servce service")
	}
}

func (s *Server) Connect(stream proto.MessageService_ConnectServer) error {

	log.Println("Server: New client connected")

	client := ClientConnection{ stream: stream }

	id := s.AddClient(&client)
	log.Printf("Server: Added client %d: %v", id, s.clients[id])

	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := stream.Recv()

		if err != nil {
			switch err {
			case io.EOF: 
					log.Println("Server: Got EOF from client, terminating connection")
			case context.Canceled:
					log.Println("Server: Client cancelled connection")
			default:
					log.Printf("Server: Error on recv - %s\n", err.Error())
			}
			break
		}

		log.Printf("Server: Got msg '%s' from %d\n", req.Msg, id)

		resp := proto.Message{Msg: fmt.Sprintf("From client %d: %s", id, req.Msg), Timestamp: 0}

		s.Replicate(id, &resp)

	}

	s.RemoveClient(id)

	log.Printf("Server: Client connection terminated")

	return nil
}

func main() {
	server := NewServer()
	server.StartServer()
}

