package main;

import (
	proto "ChitChat/grpc"
	"context"
	"log"
	"io"
	clocks "ChitChat/clocks"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ReceivedMessage struct {
	author uint32
	message string
	lamportTimestamp uint32
}

type Client struct {
	conn *grpc.ClientConn
	client proto.MessageServiceClient
	stream grpc.BidiStreamingClient[proto.Package, proto.Package]
	clock clocks.LamportClock
	id uint32

	callback func(ReceivedMessage, error)
}

func NewClient(ip string, port string) *Client {

	conn, err := grpc.NewClient(
		ip + ":" + port,
		grpc.WithTransportCredentials(insecure.NewCredentials()));

	if err != nil {
		log.Fatalf("Client: Failed to connect with err {%s}", err.Error());
	}

	client := proto.NewMessageServiceClient(conn)

//	ctx, cf := context.WithTimeout(context.Background(), 100 * time.Millisecond)
	stream, err := client.Connect(context.Background())


	if err != nil {
		log.Fatalf("Client: Failed to establish stream - {%s}", err.Error())
	}

	log.Printf("A\n")
	idPackage, err := stream.Recv()
	log.Printf("B\n")

	if err != nil {
		log.Fatalf("Client: Failed to recv acception package from server - {%s}", err.Error())
	}


	var id uint32
	switch v := idPackage.PackageData.(type) {
	case *proto.Package_Accepted:
		id = v.Accepted.AuthorID	
	case *proto.Package_Msg:
		log.Fatalln("Client: Got message package before accepted package")
	default:
		log.Fatalln("Client: Could not determine type of server's package response")
	}

	newClient := new(Client)
	*newClient = Client { 
		conn: conn,
		client: client,
		stream: stream,
		clock: clocks.NewClock(0),
		id: id,
		callback: func (ReceivedMessage, error) { println("Client: Unhandled callback") },
	}

	go newClient.msgHandler()

	return newClient
}

func (this *Client) Send(message string) error {
	this.clock.ProgressTime()
	err := this.stream.Send(
		&proto.Package{
			PackageData: &proto.Package_Msg{Msg: &proto.Message{AuthorID: this.id, Msg: message} }, 
			MetaData: &proto.MetaData {Timestamp: this.clock.ThisTime()} })


	return err
}

func (this *Client) Recv() (ReceivedMessage, error) {
	resp, err := this.stream.Recv()

	if err == io.EOF {
		this.stream.CloseSend()
		this.clock.ProgressTime()
		return ReceivedMessage {0, "", this.clock.ThisTime()}, err
	} else if resp == nil {
		return ReceivedMessage {0, "", this.clock.ThisTime()}, err
	}

	clock := clocks.NewClock(resp.MetaData.Timestamp)
	this.clock.MergeClocks(&clock)

	// TODO, ensure package type is correct
	return ReceivedMessage {resp.GetMsg().AuthorID, resp.GetMsg().Msg, this.clock.ThisTime()}, nil
		
}

func (this *Client) msgHandler() {
	for {
		resp, err := this.Recv()
		for this.callback == nil {}
		this.callback(resp, err)
	}
}

func (this *Client) Close() {
	this.stream.CloseSend()
}

func (this *Client) Id() uint32 {
	return this.id
}

func (this *Client) SetCallback(callback func(ReceivedMessage, error)) {
	this.callback = callback
}

