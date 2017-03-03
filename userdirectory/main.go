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
	"flag"
	"net"
	"os"

	"cloud.google.com/go/datastore"

	pb "github.com/ahmetb/coffeelog/coffeelog"
	"github.com/ahmetb/coffeelog/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	projectID = flag.String("google-project-id", "", "google cloud project id")
	addr      = flag.String("addr", ":8001", "[host]:port to listen")

	log *logrus.Entry
)

func main() {
	flag.Parse()
	host, err := os.Hostname()
	if err != nil {
		log.Fatal(errors.Wrap(err, "cannot get hostname"))
	}
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	log = logrus.WithFields(logrus.Fields{
		"service": "userdirectory",
		"host":    host,
		"v":       version.Version(),
	})

	if env := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); env == "" {
		log.Fatal("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
	}

	if *projectID == "" {
		log.Fatal("google cloud project id is not set")
	}

	ctx := context.Background()
	ds, err := datastore.NewClient(ctx, *projectID)
	if err != nil {
		log.WithField("error", err).Fatal("failed to create client")
	}
	defer ds.Close()

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterUserDirectoryServer(grpcServer, &userDirectory{ds})
	log.WithField("addr", *addr).Info("starting to listen on grpc")
	log.Fatal(grpcServer.Serve(lis))
}
