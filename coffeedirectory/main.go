package main

import (
	"context"
	"net"

	"cloud.google.com/go/datastore"
	pb "github.com/ahmetalpbalkan/coffeelog/coffeelog"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	projectID = "ahmetb-starter" // TODO configurable

	coffeeDirectoryBackend = "127.0.0.1:8002" // TODO use service discovery
	userDirectoryBackend   = "127.0.0.1:8001" // TODO use service discovery
)

var log *logrus.Entry

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log = logrus.WithField("service", "coffeedirectory")

	ds, err := datastore.NewClient(context.TODO(), projectID)
	if err != nil {
		log.WithField("error", err).Fatal("failed to create client")
	}
	defer ds.Close()

	addr := "127.0.0.1:8002" // TODO make configurable
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	svc := &service{ds}
	pb.RegisterRoasterDirectoryServer(grpcServer, svc)
	pb.RegisterActivityDirectoryServer(grpcServer, svc)
	log.WithField("addr", addr).WithField("service", "coffeedirectory").Info("starting to listen on grpc")
	log.Fatal(grpcServer.Serve(lis))
}
