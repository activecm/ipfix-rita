package integrationtest

import (
	"context"
	"fmt"
	"testing"

	"github.com/activecm/dbtest"
	"github.com/activecm/dbtest/docker"
	"github.com/activecm/ipfix-rita/converter/environment"
)

//Dependencies holds the dependencies needed for conducting
//an integration test
type Dependencies struct {
	loader         docker.Loader
	mongoDB        dbtest.MongoDBContainer
	env            environment.Environment
	envInitialized bool
}

//dependencies is a singleton of Dependencies
//this allows the dependencies to be reused across invocations
//of GetDependencies
var dependencies *Dependencies

//GetDependencies returns a singleton instance of Dependencies.
//If the short flag is supplied to Go test, GetDependencies
//will cause the test to be skipped
func GetDependencies(t *testing.T) *Dependencies {
	if testing.Short() {
		t.Skip()
	}
	if dependencies == nil {
		loader, mongoDB, err := newMongoDBContainer()
		if err != nil {
			fmt.Printf("%+v\n", err)
			t.FailNow()
		}
		dependencies = &Dependencies{loader: loader, mongoDB: mongoDB}
	}

	return dependencies
}

//CloseDependencies tears down the dependencies created
//through GetDependencies. There is no effect if
//GetDependencies was never called.
func CloseDependencies() {
	if dependencies != nil {
		if dependencies.envInitialized {
			dependencies.env.DB.Close()
		}
		dependencies.loader.StopService(context.Background(), dependencies.mongoDB)
		dependencies.loader.Close()
		dependencies = nil
	}
}

//GetFreshEnvironment returns a singleton Environment tailored for testing.
//This method clears out any data in the collections specified in
//Environment.DB. As such, it is not thread-safe.
func (deps *Dependencies) GetFreshEnvironment(t *testing.T) environment.Environment {
	if !deps.envInitialized {
		deps.env = newEnvironment(t, deps.mongoDB.GetMongoDBURI())
		deps.envInitialized = true
	}

	//clear out any old data
	inputColl := deps.env.DB.NewInputConnection()
	_, err := inputColl.RemoveAll(nil)
	if err != nil {
		deps.env.Error(err, nil)
		t.FailNow()
	}
	inputColl.Database.Session.Close()

	outputColl, err := deps.env.DB.NewOutputConnection("")
	if err != nil {
		deps.env.Error(err, nil)
		t.FailNow()
	}
	_, err = outputColl.RemoveAll(nil)
	if err != nil {
		deps.env.Error(err, nil)
		t.FailNow()
	}
	outputColl.Database.Session.Close()

	sessAggColl := deps.env.DB.NewSessionsConnection()
	_, err = sessAggColl.RemoveAll(nil)
	if err != nil {
		deps.env.Error(err, nil)
		t.FailNow()
	}
	sessAggColl.Database.Session.Close()

	dbsColl := deps.env.DB.NewMetaDBDatabasesConnection()
	_, err = dbsColl.RemoveAll(nil)
	if err != nil {
		deps.env.Error(err, nil)
		t.FailNow()
	}
	dbsColl.Database.Session.Close()
	return deps.env
}
