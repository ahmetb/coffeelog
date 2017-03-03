// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"net"
	"os"

	"flag"

	"cloud.google.com/go/datastore"
	pb "github.com/ahmetb/coffeelog/coffeelog"
	"github.com/ahmetb/coffeelog/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
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
	logrus.SetFormatter(&logrus.JSONFormatter{FieldMap: logrus.FieldMap{logrus.FieldKeyLevel: "severity"}})
	log = logrus.WithFields(logrus.Fields{
		"service": "coffeedirectory",
		"host":    host,
		"v":       version.Version(),
	})
	grpclog.SetLogger(log.WithField("facility", "grpc"))

	if env := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); env == "" {
		log.Fatal("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
	}
	if *userDirectoryBackend == "" {
		log.Fatal("user directory flag not specified")
	}
	if *projectID == "" {
		log.Fatal("google cloud project id is not set")
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
