package main

import (
	"fmt"

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
		"op":        "authz",
		"google.id": goog.GetID()})
	log.Debug("received request")

	ds, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		log.WithField("error", err).Fatal("failed to create client")
		return nil, errors.New("failed to initialize database client")
	}

	q := datastore.NewQuery("Account").Filter("GoogleID =", goog.ID).Limit(1)
	var v []account
	if _, err := ds.GetAll(ctx, q, &v); err != nil {
		log.WithField("error", err).Error("failed to query the datastore")
		return nil, errors.New("failed to query")
	}
	user := &pb.User{

		DisplayName: v[0].DisplayName,
		Picture:     v[0].Picture,
	}
	if len(v) == 0 {
		// create new account
		k, err := ds.Put(ctx, datastore.IncompleteKey("Account", nil), account{
			Email:       goog.Email,
			DisplayName: goog.DisplayName,
			Picture:     goog.PictureURL,
			GoogleID:    goog.ID,
		})
		if err != nil {
			log.WithField("error", err).Error("failed to save to datastore")
			return nil, errors.New("failed to save")
		}
		user.ID = fmt.Sprintf("%d", k.ID)
		log.WithField("id", user.ID).Info("created new user account")
	} else {
		// return existing account
		user.ID = fmt.Sprintf("%d", v[0].K.ID)
		log.WithField("id", user.ID).Debug("user exists")
	}
	return user, nil
}
