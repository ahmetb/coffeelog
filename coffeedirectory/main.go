package main

import (
	"context"
	"net"
	"os"

	"cloud.google.com/go/datastore"
	pb "github.com/ahmetalpbalkan/coffeelog/coffeelog"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	projectID = "ahmetb-starter" // TODO make configurable
)

var (
	userDirectoryBackend string
	log                  *logrus.Entry
)

func main() {
	host, err := os.Hostname()
	if err != nil {
		log.Fatal(errors.Wrap(err, "cannot get hostname"))
	}
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	log = logrus.WithFields(logrus.Fields{
		"service": "coffeedirectory",
		"host":    host,
	})

	if env := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); env == "" {
		log.Fatal("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
	}
	userDirectoryBackend = os.Getenv("USER_DIRECTORY_HOST")
	if userDirectoryBackend == "" {
		log.Fatal("USER_DIRECTORY_HOST not set")
	}

	ds, err := datastore.NewClient(context.TODO(), projectID)
	if err != nil {
		log.WithField("error", err).Fatal("failed to create client")
	}
	defer ds.Close()

	addr := "0.0.0.0:8002" // TODO make configurable
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	cc, err := grpc.Dial(userDirectoryBackend, grpc.WithInsecure())
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to contact user directory"))
	}
	defer func() {
		log.Debug("closing connection to user directory")
		cc.Close()
	}()

	grpcServer := grpc.NewServer()
	svc := &service{ds, pb.NewUserDirectoryClient(cc)}
	pb.RegisterRoasterDirectoryServer(grpcServer, svc)
	pb.RegisterActivityDirectoryServer(grpcServer, svc)
	log.WithField("addr", addr).WithField("service", "coffeedirectory").Info("starting to listen on grpc")
	log.Fatal(grpcServer.Serve(lis))
}
