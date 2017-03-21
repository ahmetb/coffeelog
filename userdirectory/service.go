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
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/trace"
	pb "github.com/ahmetb/coffeelog/coffeelog"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type userDirectory struct {
	ds *datastore.Client
}

type account struct {
	K           *datastore.Key `datastore:"__key__"`
	DisplayName string         `datastore:"DisplayName"`
	Email       string         `datastore:"Email"`
	Picture     string         `datastore:"Picture"`
	GoogleID    string         `datastore:"GoogleID"`
}

func (u *userDirectory) AuthorizeGoogle(ctx context.Context, goog *pb.GoogleUser) (*pb.User, error) {
	span := trace.FromContext(ctx).NewChild("usersvc/AuthorizeGoogle")
	defer span.Finish()

	log := log.WithFields(logrus.Fields{
		"op":        "AuthorizeGoogle",
		"google.id": goog.GetID()})
	log.Debug("received request")

	cs := span.NewChild("datastore/query/account/by_googleid")
	q := datastore.NewQuery("Account").Filter("GoogleID =", goog.ID).Limit(1)
	var v []account
	if _, err := u.ds.GetAll(ctx, q, &v); err != nil {
		log.WithField("error", err).Error("failed to query the datastore")
		return nil, errors.Wrap(err, "failed to query")
	}
	cs.Finish()

	var id string
	if len(v) == 0 {
		cs = span.NewChild("datastore/put/account")
		// create new account
		k, err := u.ds.Put(ctx, datastore.IncompleteKey("Account", nil), &account{
			Email:       goog.Email,
			DisplayName: goog.DisplayName,
			Picture:     goog.PictureURL,
			GoogleID:    goog.ID,
		})
		if err != nil {
			log.WithField("error", err).Error("failed to save to datastore")
			return nil, errors.New("failed to save")
		}
		id = fmt.Sprintf("%d", k.ID)
		log.WithField("id", id).Info("created new user account")
		cs.Finish()
	} else {
		// return existing account
		id = fmt.Sprintf("%d", v[0].K.ID)
		log.WithField("id", id).Debug("user exists")
	}

	// retrieve user again from backend
	user, err := u.GetUser(ctx, &pb.UserRequest{ID: id})
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve user")
	} else if !user.GetFound() {
		return nil, errors.New("cannot find user that is just created")
	}
	return user.GetUser(), nil
}

func (u *userDirectory) GetUser(ctx context.Context, req *pb.UserRequest) (*pb.UserResponse, error) {
	span := trace.FromContext(ctx).NewChild("usersvc/GetUser")
	defer span.Finish()

	log := log.WithFields(logrus.Fields{
		"op": "GetUser",
		"id": req.GetID()})
	start := time.Now()
	defer func() {
		log.WithField("elapsed", time.Since(start).String()).Debug("completed request")
	}()
	log.Debug("received request")

	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		return nil, errors.New("cannot parse ID")
	}

	cs := span.NewChild("datastore/query/account/by_id")
	defer cs.Finish()

	var v account
	err = u.ds.Get(ctx, datastore.IDKey("Account", id, nil), &v)
	if err == datastore.ErrNoSuchEntity {
		log.Debug("user not found")
		return &pb.UserResponse{Found: false}, nil
	} else if err != nil {
		log.WithField("error", err).Error("failed to query the datastore")
		return nil, errors.Wrap(err, "failed to query")
	}
	log.Debug("found user")
	return &pb.UserResponse{
		Found: true,
		User: &pb.User{
			ID:          fmt.Sprintf("%d", v.K.ID),
			DisplayName: v.DisplayName,
			Picture:     v.Picture}}, nil
}
