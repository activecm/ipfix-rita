package integrationtest

import (
	"context"

	"github.com/activecm/dbtest"
	"github.com/activecm/dbtest/docker"
)

func newMongoDBContainer() (docker.Loader, dbtest.MongoDBContainer, error) {
	loader, err := docker.NewLoader()
	if err != nil {
		return docker.Loader{}, dbtest.MongoDBContainer{}, err
	}
	mongo, err := dbtest.NewMongoDBContainer(context.Background(), loader)
	if err != nil {
		return docker.Loader{}, dbtest.MongoDBContainer{}, err
	}
	return loader, mongo, nil
}
