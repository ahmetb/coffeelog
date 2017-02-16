package main

import (
	"fmt"

	"strconv"

	"cloud.google.com/go/datastore"
	pb "github.com/ahmetalpbalkan/coffeelog/coffeelog"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const (
	projectID = "ahmetb-starter" // TODO configurable
)

type userDirectory struct{}

type account struct {
	K           *datastore.Key `datastore:"__key__"`
	DisplayName string         `datastore:"DisplayName"`
	Email       string         `datastore:"Email"`
	Picture     string         `datastore:"Picture"`
	GoogleID    string         `datastore:"GoogleID"`
}

func (u *userDirectory) AuthorizeGoogle(ctx context.Context, goog *pb.GoogleUser) (*pb.User, error) {
	log := logrus.WithFields(logrus.Fields{
		"op":        "AuthorizeGoogle",
		"google.id": goog.GetID()})
	log.Debug("received request")

	ds, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		log.WithField("error", err).Fatal("failed to create client")
		return nil, errors.New("failed to initialize database client")
	}
	defer ds.Close()

	q := datastore.NewQuery("Account").Filter("GoogleID =", goog.ID).Limit(1)
	var v []account
	if _, err := ds.GetAll(ctx, q, &v); err != nil {
		log.WithField("error", err).Error("failed to query the datastore")
		return nil, errors.New("failed to query")
	}

	var id string
	if len(v) == 0 {
		// create new account
		k, err := ds.Put(ctx, datastore.IncompleteKey("Account", nil), &account{
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
	} else {
		// return existing account
		id = fmt.Sprintf("%d", v[0].K.ID)
		log.WithField("id", id).Debug("user exists")
	}

	// retrieve user again from backend
	user, err := u.GetUser(ctx, &pb.UserRequest{ID: id})
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve user")
	} else if !user.Found {
		return nil, errors.New("cannot find user that is just created")
	}
	return user.User, nil
}

func (u *userDirectory) GetUser(ctx context.Context, req *pb.UserRequest) (*pb.UserResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"op": "GetUser",
		"id": req.GetID()})
	log.Debug("received request")

	// TODO this block is highly duplicated, eliminate
	ds, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		log.WithField("error", err).Fatal("failed to create client")
		return nil, errors.New("failed to initialize database client")
	}
	defer ds.Close()

	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		return nil, errors.New("cannot parse ID")
	}

	var v account
	err = ds.Get(ctx, datastore.IDKey("Account", id, nil), &v)
	if err == datastore.ErrNoSuchEntity {
		log.Debug("user not found")
		return &pb.UserResponse{Found: false}, nil
	} else if err != nil {
		log.WithField("error", err).Error("failed to query the datastore")
		return nil, errors.New("failed to query")
	}
	log.Debug("found user")
	return &pb.UserResponse{
		Found: true,
		User: &pb.User{
			ID:          fmt.Sprintf("%d", v.K.ID),
			DisplayName: v.DisplayName,
			Picture:     v.Picture}}, nil
}
