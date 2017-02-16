package main

import (
	"context"
	"net"

	"cloud.google.com/go/datastore"

	pb "github.com/ahmetalpbalkan/coffeelog/coffeelog"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	projectID = "ahmetb-starter" // TODO configurable
)

func main() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})

	ds, err := datastore.NewClient(context.TODO(), projectID)
	if err != nil {
		log.WithField("error", err).Fatal("failed to create client")
	}
	defer ds.Close()

	addr := "127.0.0.1:8001" // TODO make configurable
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterUserDirectoryServer(grpcServer, &userDirectory{ds})
	log.WithField("addr", addr).Info("starting to listen on grpc")
	log.Fatal(grpcServer.Serve(lis))
}
