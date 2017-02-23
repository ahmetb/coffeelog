package main

import (
	"time"

	"cloud.google.com/go/datastore"
	pb "github.com/ahmetalpbalkan/coffeelog/coffeelog"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const (
	kindRoaster  = "Roaster"  // datastore kind
	kindActivity = "Activity" // datastore kind
)

type service struct {
	ds *datastore.Client
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

func (c *service) ListRoaster(ctx context.Context, _ *pb.RoastersRequest) (*pb.RoastersResponse, error) {
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

	// TODO upload picture

	ts, err := ptypes.Timestamp(req.GetDate())
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse date from proto")
	}
	v := activity{
		UserID:      "", // TODO fix: use the user!!!
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
		PictureURL:  "", // TODO fix
	}
	k, err := c.ds.Put(ctx, datastore.IncompleteKey(kindActivity, nil), &v)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save activity")
	}

	log.WithField("id", k.ID).Info("activity saved to datastore")
	return &pb.PostActivityResponse{ID: k.ID}, nil
}
