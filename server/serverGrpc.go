package main;

import (
	proto "ChitChat/grpc"

	clocks "ChitChat/clocks"

	"net"
	"log"
	"sync"
	"google.golang.org/grpc"

	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
)

type ClientConnection struct {
	stream proto.MessageService_ConnectServer
}

type Server struct {
	proto.UnimplementedMessageServiceServer
	clientCount int
	clients []*ClientConnection
	clock clocks.LamportClock
	mu sync.Mutex
};

func NewServer() Server {
	return Server{clientCount: 0, clock: clocks.NewClock(0)}
}

func (s *Server) AddClient(client *ClientConnection) int {
	s.mu.Lock()

	id := len(s.clients)

	client.stream.Send(
		&proto.Package{
			PackageData: &proto.Package_Accepted{Accepted: &proto.Accepted{AuthorID: uint32(id)} }, 
			MetaData: &proto.MetaData {Timestamp: s.clock.ThisTime()} })

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

func (s *Server) Replicate(id int, message *proto.Package) {
	for i := range s.clientCount {
//		if i == id { continue; }
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
			log.Println("Server: Context is done")
			return ctx.Err()
		default:
		}

		req, err := stream.Recv()

		if err != nil {
			switch status.Code(err) {
			case codes.Aborted: 
					log.Println("Server: Got EOF from client, terminating connection")
			case codes.Canceled:
					log.Println("Server: Client cancelled connection")
					continue
			default:
					log.Printf("Server: Error on recv - %s\n", err.Error())
			}
			break
		}

		switch req.PackageData.(type) {
		case *proto.Package_Accepted:
			log.Printf("Server: Client %d sent package accepted instead of message\n", id)
			default:
		}

		log.Printf("Server: Got msg '%s' from %d\n", req.GetMsg().Msg, id)

		clientClock := clocks.NewClock(req.MetaData.Timestamp)
		s.clock.MergeClocks(&clientClock)

		req.MetaData.Timestamp = s.clock.ThisTime()

		s.Replicate(id, req)
	}

	s.RemoveClient(id)

	log.Printf("Server: Client connection terminated")

	return nil
}

func main() {
	server := NewServer()
	server.StartServer()
}

