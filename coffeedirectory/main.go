package main

import (
	"context"
	"net"
	"os"

	"flag"

	"cloud.google.com/go/datastore"
	pb "github.com/ahmetalpbalkan/coffeelog/coffeelog"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	userDirectoryBackend = flag.String("user-directory-addr", "", "address of user directory backend")
	projectID            = flag.String("google-project-id", "", "google cloud project id")
	addr                 = flag.String("addr", ":8000", "[host]:port to listen")

	log *logrus.Entry
)

func main() {
	flag.Parse()
	ctx := context.Background()
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
	if *userDirectoryBackend == "" {
		log.Fatal("user directory flag not specified")
	}

	ds, err := datastore.NewClient(ctx, *projectID)
	if err != nil {
		log.WithField("error", err).Fatal("failed to create client")
	}
	defer ds.Close()

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}

	cc, err := grpc.Dial(*userDirectoryBackend, grpc.WithInsecure())
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
	log.WithFields(logrus.Fields{"addr": *addr,
		"service": "coffeedirectory"}).Info("starting to listen on grpc")
	log.Fatal(grpcServer.Serve(lis))
}
