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
	"path"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/storage"
	pb "github.com/ahmetb/coffeelog/coffeelog"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const (
	kindRoaster  = "Roaster"  // datastore kind
	kindActivity = "Activity" // datastore kind

	bucketPics = "coffeepics" // storage bucket
)

type service struct {
	ds      *datastore.Client
	userSvc pb.UserDirectoryClient
}

// roaster as represented in Datastore.
type roaster struct {
	K         *datastore.Key `datastore:"__key__"`
	Name      string         `datastore:"Name"`
	Picture   string         `datastore:"Picture,noindex"`
	CreatedBy string         `datastore:"CreatedBy"` // TODO use
}

func (r *roaster) ToProto() *pb.Roaster {
	return &pb.Roaster{
		ID:   r.K.ID,
		Name: r.Name,
	}
}

// activity as represented in Datastore.
type activity struct {
	K           *datastore.Key `datastore:"__key__"`
	UserID      string         `datastore:"UserID"`
	Date        time.Time      `datastore:"Date"`
	LogDate     time.Time      `datastore:"LogDate"`
	Drink       string         `datastore:"Drink"`
	Homebrew    bool           `datastore:"Homebrew"`
	Amount      int32          `datastore:"Amount"`
	AmountUnit  string         `datastore:"AmountUnit"`
	Method      string         `datastore:"Method"`
	Origin      string         `datastore:"Origin"`
	RoasterID   int64          `datastore:"RoasterID"`
	RoasterName string         `datastore:"RoasterName,noindex"`
	Notes       string         `datastore:"Notes,noindex"`
	PictureURL  string         `datastore:"PictureURL,noindex"`
}

func (v *activity) ToProto(u *pb.User) (*pb.Activity, error) {
	dateTs, err := ptypes.TimestampProto(v.Date)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse date from proto")
	}
	logDateTs, err := ptypes.TimestampProto(v.LogDate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse date from proto")
	}
	return &pb.Activity{
		ID:         v.K.ID,
		User:       u,
		Drink:      v.Drink,
		Method:     v.Method,
		Homebrew:   v.Homebrew,
		Origin:     v.Origin,
		PictureURL: v.PictureURL,
		Roaster: &pb.Activity_RoasterInfo{
			ID:   v.RoasterID,
			Name: v.RoasterName},
		Date:    dateTs,
		LogDate: logDateTs,
		Amount: &pb.Activity_DrinkAmount{
			N:    v.Amount,
			Unit: pb.Activity_DrinkAmount_CaffeineUnit(pb.Activity_DrinkAmount_CaffeineUnit_value[v.AmountUnit])},
		Notes: v.Notes,
	}, nil
}

func (c *service) GetRoaster(ctx context.Context, req *pb.RoasterRequest) (*pb.RoasterResponse, error) {
	e := log.WithField("q.id", req.GetID()).WithField("q.name", req.GetName())
	e.Debug("querying roaster")
	q := datastore.NewQuery(kindRoaster)
	var v []roaster
	if req.GetName() != "" {
		q = q.Filter("Name =", req.GetName())
	} else {
		q = q.Filter("__key__ =", datastore.IDKey(kindRoaster, req.GetID(), nil))
	}
	if _, err := c.ds.GetAll(ctx, q.Limit(1), &v); err != nil {
		log.WithField("error", err).Error("failed to query datastore")
		return nil, errors.New("failed to retrieve roaster")
	} else if len(v) == 0 {
		return &pb.RoasterResponse{Found: false}, nil
	}
	e.WithField("count", len(v)).Debug("results retrieved")
	return &pb.RoasterResponse{Found: true, Roaster: v[0].ToProto()}, nil
}

func (c *service) CreateRoaster(ctx context.Context, req *pb.RoasterCreateRequest) (*pb.Roaster, error) {
	k, err := c.ds.Put(ctx, datastore.IncompleteKey(kindRoaster, nil), &roaster{
		Name: req.Name})
	if err != nil {
		log.WithField("error", err).Error("failed to insert to datastore")
		return new(pb.Roaster), errors.New("failed to save the roaster")
	}

	r, err := c.GetRoaster(ctx, &pb.RoasterRequest{Query: &pb.RoasterRequest_ID{ID: k.ID}})
	if err != nil {
		log.WithField("error", err).Error("failed to query the saved roaster")
		return new(pb.Roaster), errors.New("failed to query the saved roaster")
	}
	log.WithFields(logrus.Fields{
		"id":   r.GetRoaster().GetID(),
		"name": r.GetRoaster().GetName()}).Debug("new roaster created")
	return r.GetRoaster(), nil
}

func (c *service) ListRoasters(ctx context.Context, _ *pb.RoastersRequest) (*pb.RoastersResponse, error) {
	resp := new(pb.RoastersResponse)

	var data []roaster
	if _, err := c.ds.GetAll(ctx, datastore.NewQuery(kindRoaster), &data); err != nil {
		log.WithField("error", err).Error("datastore query failed")
		return resp, errors.New("failed to retrieve roasters")
	}

	var r []*pb.Roaster
	for _, v := range data {
		r = append(r, v.ToProto())
	}
	log.WithField("count", len(r)).Debug("retrieved roasters list")
	resp.Results = r
	return resp, nil
}

func (c *service) PostActivity(ctx context.Context, req *pb.PostActivityRequest) (*pb.PostActivityResponse, error) {
	// resolve the roaster
	e := log.WithField("roaster.name", req.GetRoasterName())
	e.Debug("resolving roaster for activity")
	var roaster *pb.Roaster
	if rr, err := c.GetRoaster(ctx, &pb.RoasterRequest{Query: &pb.RoasterRequest_Name{Name: req.GetRoasterName()}}); err != nil {
		return nil, errors.New("failed to query roaster by name")
	} else if !rr.GetFound() {
		e.Debug("roaster not found, creating")
		rcr, err := c.CreateRoaster(ctx, &pb.RoasterCreateRequest{Name: req.GetRoasterName()})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create a new roaster")
		}
		roaster = rcr
		e.WithField("roaster.id", rcr.GetID()).Debug("new roaster created")
	} else {
		log.WithField("roaster.id", roaster.GetID()).Debug("using existing roaster")
		roaster = rr.GetRoaster()
	}

	var picURL string
	if req.GetPicture() != nil {
		url, err := uploadPicture(ctx, req.GetPicture().GetFilename(),
			req.GetPicture().GetContentType(),
			req.GetPicture().GetData())
		if err != nil {
			return nil, errors.Wrap(err, "failed to upload picture")
		}
		picURL = url
	}

	ts, err := ptypes.Timestamp(req.GetDate())
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse date from proto")
	}
	v := activity{
		UserID:      req.GetUserID(),
		Date:        ts,
		LogDate:     time.Now(),
		Drink:       req.GetDrink(),
		Homebrew:    req.GetHomebrew(),
		Method:      req.GetMethod(),
		Origin:      req.GetOrigin(),
		Amount:      req.GetAmount().GetN(),
		AmountUnit:  req.GetAmount().GetUnit().String(),
		RoasterID:   roaster.GetID(),
		RoasterName: roaster.GetName(),
		Notes:       req.GetNotes(),
		PictureURL:  picURL,
	}
	k, err := c.ds.Put(ctx, datastore.IncompleteKey(kindActivity, nil), &v)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save activity")
	}

	log.WithField("id", k.ID).Info("activity saved to datastore")
	return &pb.PostActivityResponse{ID: k.ID}, nil
}

func uploadPicture(ctx context.Context, filename, contentType string, b []byte) (string, error) {
	t := time.Now()
	cl, err := storage.NewClient(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to create storage client")
	}
	defer cl.Close()

	fn := fmt.Sprintf("%d/%02d/%s%s", t.Year(), t.Month(), uuid.NewV4(), path.Ext(filename))
	w := cl.Bucket(bucketPics).Object(fn).NewWriter(ctx)
	w.ContentType = contentType
	w.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}
	if _, err := w.Write(b); err != nil {
		return "", errors.Wrap(err, "failed to write to storage object")
	}
	log.WithFields(logrus.Fields{"bucket": bucketPics,
		"object": fn}).Debug("uploaded file")
	url := fmt.Sprintf("https://%s.storage.googleapis.com/%s", bucketPics, fn)
	return url, errors.Wrap(w.Close(), "failed to close object writer")
}

func (c *service) GetActivity(ctx context.Context, req *pb.ActivityRequest) (*pb.Activity, error) {
	var v activity
	if err := c.ds.Get(ctx, datastore.IDKey(kindActivity, req.GetID(), nil), &v); err == datastore.ErrNoSuchEntity {
		return nil, errors.New("activity not found")
	} else if err != nil {
		return nil, errors.Wrap(err, "error querying datastore for activity")
	}

	user, err := c.userSvc.GetUser(ctx, &pb.UserRequest{ID: v.UserID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve activity owner")
	}
	if !user.GetFound() {
		return nil, errors.Wrap(err, "activity owner does not exist")
	}

	activity, err := v.ToProto(user.GetUser())
	return activity, errors.Wrap(err, "activity proto conversion failed")
}

func (c *service) GetUserActivities(ctx context.Context, req *pb.UserActivitiesRequest) (*pb.UserActivitiesResponse, error) {
	log.WithField("user.id", req.GetUserID()).Debug("querying datastore for activities")

	user, err := c.userSvc.GetUser(ctx, &pb.UserRequest{ID: req.GetUserID()})
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve user profile")
	}
	if !user.GetFound() {
		return nil, errors.Wrap(err, "user does not exist")
	}

	var v []activity
	if _, err := c.ds.GetAll(ctx, datastore.NewQuery(kindActivity).Filter("UserID =", req.GetUserID()).Order("-Date"), &v); err != nil {
		return nil, errors.Wrap(err, "failed to query datastore for user activities")
	}

	var res []*pb.Activity
	for _, a := range v {
		aa, err := a.ToProto(user.GetUser())
		if err != nil {
			return nil, errors.Wrap(err, "proto conversion failed on one of the activities")
		}
		res = append(res, aa)
	}
	return &pb.UserActivitiesResponse{Activities: res}, nil
}
