package main;

import (
	proto "ChitChat/grpc"
	"context"
	"log"
	"io"
	clocks "ChitChat/logical_clocks"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type MessageKind uint8

const (
	MessageEvent MessageKind = iota
	LoginEvent 
	LogoutEvent
	ErrEvent
)

type ReceivedMessage struct {
	event MessageKind
	author string
	message string
	lamportTimestamp uint64
}

type Client struct {
	conn *grpc.ClientConn
	client proto.ChitChatServiceClient
	stream grpc.BidiStreamingClient[proto.StreamRequest, proto.StreamResponse]
	clock *clocks.LamportClock
	username string

	callback func(ReceivedMessage, error)
}

func NewClient(ip string, port string, username string) *Client {

	conn, err := grpc.NewClient(
		ip + ":" + port,
		grpc.WithTransportCredentials(insecure.NewCredentials()));

	if err != nil {
		log.Fatalf("Client: Failed to connect with err {%s}", err.Error());
	}

	clock := clocks.NewLamport()
	clock.Tick()

	client := proto.NewChitChatServiceClient(conn)

	resp, err := client.Connect(context.Background(),
		&proto.ConnectRequest {Username: username, Timestamp: clock.Now()})

	if err != nil {
		conn.Close()
		return nil
	}

	token := resp.GetToken()
	serverTimestamp := resp.GetTimestamp()

	clock.Sync(clocks.From(serverTimestamp))

	md := metadata.New(map[string]string{"authorization": token})

	ctx := context.Background()
	ctxWithMetaData := metadata.NewOutgoingContext(ctx, md)

	stream, err := client.Stream(ctxWithMetaData)

	newClient := new(Client)
	*newClient = Client { 
		conn: conn,
		client: client,
		stream: stream,
		username: username,
		clock: clock,
		callback: func (ReceivedMessage, error) { println("Client: Unhandled callback") },
	}

	go newClient.msgHandler()

	return newClient
}

func (this *Client) Send(message string) error {
	this.clock.Tick()
	err := this.stream.Send(
		&proto.StreamRequest{
			Timestamp: this.clock.Now(),
			Message: message,
	})

	return err
}

func (this *Client) recv() (ReceivedMessage, error) {
	resp, err := this.stream.Recv()
	this.clock.Tick()

	if err == io.EOF {
		this.stream.CloseSend()
		return ReceivedMessage {ErrEvent, "", "", this.clock.Now()}, err
	} else if resp == nil {
		return ReceivedMessage {ErrEvent, "", "", this.clock.Now()}, err
	}

	this.clock.Sync(clocks.From(resp.Timestamp))

	msg := ReceivedMessage {};
	switch ev := resp.Event.(type) {
	case *proto.StreamResponse_ChatMessage:
	msg.event = MessageEvent
	msg.message = ev.ChatMessage.Message
	msg.author = ev.ChatMessage.Username
	case *proto.StreamResponse_LoginEvent:
	msg.event = LoginEvent
	msg.author = ev.LoginEvent.Username
	case *proto.StreamResponse_LogoutEvent:
	msg.event = LogoutEvent
	msg.author = ev.LogoutEvent.Username
	}

	msg.lamportTimestamp = this.clock.Now()

	if msg.author == this.Username() {
		msg.author = "You"
	}

	return msg, nil
}

func (this *Client) msgHandler() {
	for {
		resp, err := this.recv()
		for this.callback == nil {}
		this.callback(resp, err)
	}
}

func (this *Client) Close() {
	this.stream.CloseSend()
}

func (this *Client) Username() string {
	return this.username
}

func (this *Client) SetCallback(callback func(ReceivedMessage, error)) {
	this.callback = callback
}

