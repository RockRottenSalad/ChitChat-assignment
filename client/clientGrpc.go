package main;

import (
	proto "ChitChat/grpc"
	"context"
	"time"
	"log"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn *grpc.ClientConn
	client proto.MessageServiceClient
	stream grpc.BidiStreamingClient[proto.Message, proto.Message]

}

func NewClient(ip string, port string) Client {

	conn, err := grpc.NewClient(
		ip + ":" + port,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(time.Second));

	if err != nil {
		log.Fatalf("Client: Failed to connect with err {%s}", err.Error());
	}

	client := proto.NewMessageServiceClient(conn)

	stream, err := client.Connect(context.Background())

	if err != nil {
		log.Fatalf("Client: Failed to establish stream - {%s}", err.Error())
	}

	return Client { conn: conn, client: client, stream: stream}
}

func (c *Client) Send(message string) error {
	err := c.stream.Send(&proto.Message{Msg: message, Timestamp: 0})

	return err
}

func (c *Client) Recv() (string, error) {
	resp, err := c.stream.Recv()

	if err == io.EOF {
		c.stream.CloseSend()
		return "", err
	}

	return resp.Msg, nil
}

func (c *Client) Close() {
	c.stream.CloseSend()
}

