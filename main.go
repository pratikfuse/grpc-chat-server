package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pratikfuse/grpc-chat-server/lib/database"
	"github.com/pratikfuse/grpc-chat-server/protobufs"
)

const PORT = 9000
const HOST = "0.0.0.0"

func SignalContext(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Println("listening for shutdown signal")
		<-sigs
		fmt.Println("shutdown signal received")
		signal.Stop(sigs)
		close(sigs)
		cancel()
	}()

	return ctx
}

func main() {

	errChan := make(chan error)

	err := database.CreateConnection()

	if err != nil {
		log.Fatal(err)
	}
	chatServer := &ChatServer{
		Host:          HOST,
		Port:          PORT,
		AccessKey:     "",
		BroadcastChan: make(chan *protobufs.ChatResponse, 1000),
		Clients:       make(map[string]chan *protobufs.ChatResponse),
		Mutex:         sync.RWMutex{},
	}

	signalCtx := SignalContext(context.Background())

	chatServer.Start(signalCtx, errChan)

	fmt.Println("server started")

	<-errChan

}
