package main

import (
	"cloud.google.com/go/datastore"
	pb "github.com/ahmetalpbalkan/coffeelog/coffeelog"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type coffeeDirectory struct {
	ds *datastore.Client
}

// roaster as represented in Datastore.
type roaster struct {
	K         *datastore.Key `datastore:"__key__"`
	Name      string         `datastore:"Name"`
	Picture   string         `datastore:"Picture"`
	CreatedBy string         `datastore:"CreatedBy"`
}

func (r *roaster) ToProto() *pb.Roaster {
	return &pb.Roaster{
		ID:   r.K.ID,
		Name: r.Name,
	}
}

func (c *coffeeDirectory) Get(ctx context.Context, req *pb.RoasterRequest) (*pb.RoasterResponse, error) {
	q := datastore.NewQuery("roaster")
	var v []roaster
	if req.GetName() != "" {
		q = q.Filter("Name =", req.GetName())
	} else {
		q = q.Filter("__key__ =", req.GetID())
	}
	if _, err := c.ds.GetAll(ctx, q.Limit(1), &v); err == datastore.ErrNoSuchEntity {
		return &pb.RoasterResponse{Found: false}, nil
	} else if err != nil {
		log.WithField("error", err).Error("failed to query datastore")
		return nil, errors.New("failed to retrieve roaster")
	}
	return &pb.RoasterResponse{Found: true, Roaster: v[0].ToProto()}, nil
}

func (c *coffeeDirectory) Create(ctx context.Context, req *pb.RoasterCreateRequest) (*pb.Roaster, error) {
	k, err := c.ds.Put(ctx, nil, roaster{
		Name: req.Name})
	if err != nil {
		log.WithField("error", err).Error("failed to insert to datastore")
		return new(pb.Roaster), errors.New("failed to save the roaster")
	}

	r, err := c.Get(ctx, &pb.RoasterRequest{Query: &pb.RoasterRequest_ID{ID: k.ID}})
	if err != nil {
		log.WithField("error", err).Error("failed to query the saved roaster")
		return new(pb.Roaster), errors.New("failed to query the saved roaster")
	}
	log.WithFields(log.Fields{
		"id":   r.GetRoaster().GetID(),
		"name": r.GetRoaster().GetName()}).Debug("new roaster created")
	return r.GetRoaster(), nil
}

func (c *coffeeDirectory) List(ctx context.Context, _ *pb.RoastersRequest) (*pb.RoastersResponse, error) {
	resp := new(pb.RoastersResponse)

	var data []roaster
	if _, err := c.ds.GetAll(ctx, datastore.NewQuery("Roaster"), &data); err != nil {
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
