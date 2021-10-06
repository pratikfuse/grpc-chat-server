package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/pratikfuse/grpc-chat-server/protobufs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ChatServer struct {
	Port            int
	Host, AccessKey string
	BroadcastChan   chan *protobufs.ChatResponse
	Clients         map[string]chan *protobufs.ChatResponse
	Mutex           sync.RWMutex
	User            interface{}
	protobufs.UnimplementedChatServer
}

func (cs *ChatServer) Start(ctx context.Context, errChan chan<- error) {
	cs.User = nil

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	srv := grpc.NewServer()

	protobufs.RegisterChatServer(srv, cs)

	listener, err := net.Listen("tcp", cs.Host+":"+strconv.Itoa(cs.Port))

	if err != nil {
		errChan <- err
	}

	go cs.Broadcast(ctx)

	go func() {
		fmt.Println("Starting grpc server")
		if err = srv.Serve(listener); err != nil {
			errChan <- err
		}
	}()

	<-ctx.Done()

	cs.BroadcastChan <- &protobufs.ChatResponse{
		Timestamp: time.Now().String(),
		ChatEvent: &protobufs.ChatResponse_Shutdown{
			Shutdown: &protobufs.ChatResponse_ShutdownEvent{},
		},
	}

	close(cs.BroadcastChan)

	srv.GracefulStop()

	fmt.Println("closed server")
	errChan <- nil
}

func (cs *ChatServer) Chat(server protobufs.Chat_ChatServer) error {

	mData, ok := metadata.FromIncomingContext(server.Context())

	if !ok {
		return status.Error(codes.Unauthenticated, "failed to get user credentials")
	}

	username := mData["username"][0]

	go cs.initializeClientStream(username, server)

	for {
		req, err := server.Recv()

		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		fmt.Printf("%s: %s", username, req.Message)

		cs.BroadcastChan <- &protobufs.ChatResponse{
			Timestamp: time.Now().String(),
			ChatEvent: &protobufs.ChatResponse_ClientMessage{
				ClientMessage: &protobufs.ChatResponse_ChatMessageEvent{
					Name:    username,
					Message: req.Message,
				},
			},
		}
	}
	return nil
}

func (cs *ChatServer) initializeClientStream(username string, server protobufs.Chat_ChatServer) {

	stream := cs.openClientChatStream(username)
	defer cs.closeClientChatStream(username)

	for {
		select {
		case <-server.Context().Done():
			return
		case res := <-stream:
			if s, ok := status.FromError(server.Send(res)); ok {

				switch s.Code() {
				case codes.OK:
				case codes.Unavailable:
					fallthrough
				case codes.Canceled:
					fallthrough
				case codes.DeadlineExceeded:
					fmt.Println("closed connection")
					return
				default:
					fmt.Println("failed to send message")
					return
				}
			}
		}
	}

}

func (cs *ChatServer) openClientChatStream(username string) (stream chan *protobufs.ChatResponse) {

	chatResponseStream := make(chan *protobufs.ChatResponse, 100)

	cs.Mutex.Lock()
	cs.Clients[username] = chatResponseStream
	cs.Mutex.Unlock()
	return chatResponseStream

}

func (cs *ChatServer) closeClientChatStream(username string) {

	cs.Mutex.Lock()

	if stream, ok := cs.Clients[username]; ok {
		delete(cs.Clients, username)
		close(stream)
	}
	cs.Mutex.Unlock()
}

func (cs *ChatServer) Broadcast(_ context.Context) {

	for res := range cs.BroadcastChan {
		cs.Mutex.RLock()
		for username, stream := range cs.Clients {
			if messageEvent := res.GetClientMessage(); messageEvent != nil && messageEvent.Name == username {
				continue
			}
			select {
			case stream <- res:
				fmt.Println("sent to client")
			default:
				fmt.Println("client stream full")
			}
		}
		cs.Mutex.RUnlock()
	}
}
