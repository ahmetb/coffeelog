package main

import (
	"net"

	"google.golang.org/grpc"

	pb "github.com/ahmetalpbalkan/coffeelog/coffeelog"
	log "github.com/sirupsen/logrus"
)

func main() {
	addr := "127.0.0.1:8001"
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterUserDirectoryServer(grpcServer, nil) // TODO fix
	log.Fatal(grpcServer.Serve(lis))
}
