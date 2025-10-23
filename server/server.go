// server.go
package main

import (
	"context"
	"io"
	"log"
	"net"
	"sync"

	chitchatpb "itu_chitchat/grpc"
	"itu_chitchat/lamport"

	"google.golang.org/grpc"
)

type client struct {
	stream chitchatpb.Chat_ChatStreamServer
	send   chan *chitchatpb.ChatReply
	user   string
}

func (c *client) runWriter(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.stream.Send(m); err != nil {
				return
			}
		}
	}
}

type chatSrv struct {
	chitchatpb.UnimplementedChatServer

	mu      sync.RWMutex
	clients map[*client]struct{}

	clk *lamport.Clock
}

func newChatSrv() *chatSrv {
	return &chatSrv{
		clients: make(map[*client]struct{}),
		clk:     &lamport.Clock{},
	}
}

func (s *chatSrv) add(c *client) {
	s.mu.Lock()
	s.clients[c] = struct{}{}
	s.mu.Unlock()
}

func (s *chatSrv) remove(c *client) {
	s.mu.Lock()
	delete(s.clients, c)
	s.mu.Unlock()
	close(c.send)
}

func (s *chatSrv) broadcast(m *chitchatpb.ChatReply) {
	s.mu.RLock()
	for c := range s.clients {
		select {
		case c.send <- m:
		default:
			go s.remove(c)
		}
	}
	s.mu.RUnlock()
}

func (s *chatSrv) ChatStream(st chitchatpb.Chat_ChatStreamServer) error {
	ctx := st.Context()

	c := &client{
		stream: st,
		send:   make(chan *chitchatpb.ChatReply, 32),
	}
	s.add(c)
	defer s.remove(c)

	go c.runWriter(ctx)

	for {
		msg, err := st.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		s.clk.Observe(msg.GetLamport())
		c.user = msg.GetUser()

		ts := s.clk.Tick()
		s.broadcast(&chitchatpb.ChatReply{
			User:     msg.GetUser(),
			Text:     msg.GetText(),
			Lamport:  ts,
			ClientId: msg.GetClientId(),
		})
	}
}

func main() {
	lis, err := net.Listen("tcp", ":5050")
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	chitchatpb.RegisterChatServer(s, newChatSrv())
	log.Println("chitchat server listening on :5050")
	log.Fatal(s.Serve(lis))
}
