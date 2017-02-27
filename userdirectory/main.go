package main

import (
	"context"
	"net"
	"os"

	"cloud.google.com/go/datastore"

	pb "github.com/ahmetalpbalkan/coffeelog/coffeelog"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	projectID = "ahmetb-starter" // TODO configurable
)

var log *logrus.Entry

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	log = logrus.WithField("service", "userdirectory")

	if env := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); env == "" {
		log.Fatal("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
	}

	ds, err := datastore.NewClient(context.TODO(), projectID)
	if err != nil {
		log.WithField("error", err).Fatal("failed to create client")
	}
	defer ds.Close()

	addr := "0.0.0.0:8001" // TODO make configurable
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterUserDirectoryServer(grpcServer, &userDirectory{ds})
	log.WithField("addr", addr).Info("starting to listen on grpc")
	log.Fatal(grpcServer.Serve(lis))
}
