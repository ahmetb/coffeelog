package main

import (
	"net"

	pb "github.com/ahmetalpbalkan/coffeelog/coffeelog"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	log.SetLevel(log.DebugLevel)

	addr := "127.0.0.1:8001" // TODO make configurable
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterUserDirectoryServer(grpcServer, &userDirectory{})
	log.WithField("addr", addr).Info("starting to listen on grpc")
	log.Fatal(grpcServer.Serve(lis))
}
