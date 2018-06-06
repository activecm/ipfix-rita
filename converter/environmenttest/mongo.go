package environmenttest

import (
	"context"
	"testing"

	"github.com/activecm/dbtest"
	"github.com/activecm/dbtest/docker"
)

//NewMongoDBContainer connects to Docker to create a new MongoDB Container
//for use in integration testing.
func NewMongoDBContainer(t *testing.T) (docker.Loader, dbtest.MongoDBContainer) {
	if testing.Short() {
		t.Skip()
	}
	loader, err := docker.NewLoader()
	if err != nil {
		t.Errorf("%+v\n", err)
		t.FailNow()
	}
	mongo, err := dbtest.NewMongoDBContainer(context.Background(), loader)
	if err != nil {
		t.Errorf("%+v\n", err)
		t.FailNow()
	}
	return loader, mongo
}
