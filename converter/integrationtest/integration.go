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
	loader  docker.Loader
	mongoDB dbtest.MongoDBContainer
	Env     environment.Environment
}

//dependencies is a singleton of Dependencies
//this allows the dependencies to be reused across invocations
//of GetDependencies
var dependencies *Dependencies

var dependenciesResetFuncs []func(*testing.T, *Dependencies)

//GetDependencies returns a singleton instance of Dependencies.
//If the short flag is supplied to Go test, GetDependencies
//will cause the test to be skipped. Any functions registered
//with RegisterDependenciesResetFunc will be called before
//the singleton is returned.
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
		dependencies = &Dependencies{
			loader:  loader,
			mongoDB: mongoDB,
			Env:     newEnvironment(t, mongoDB.GetMongoDBURI()),
		}

		RegisterDependenciesResetFunc(resetInputColl)
		RegisterDependenciesResetFunc(resetOutputColl)
		RegisterDependenciesResetFunc(resetMetaDBDatabasesColl)
	}

	for i := range dependenciesResetFuncs {
		dependenciesResetFuncs[i](t, dependencies)
	}

	return dependencies
}

//RegisterDependenciesResetFunc registers a function
//which resets the Dependencies object to a fresh state
//when GetDependencies is called
func RegisterDependenciesResetFunc(resetFunc func(*testing.T, *Dependencies)) {
	dependenciesResetFuncs = append(dependenciesResetFuncs, resetFunc)
}

//CloseDependencies tears down the dependencies created
//through GetDependencies. There is no effect if
//GetDependencies was never called.
func CloseDependencies() {
	if dependencies != nil {
		dependencies.Env.DB.Close()
		dependencies.loader.StopService(context.Background(), dependencies.mongoDB)
		dependencies.loader.Close()
		dependencies = nil
	}
}

func resetInputColl(t *testing.T, deps *Dependencies) {
	//clear out any old data
	inputColl := deps.Env.DB.NewInputConnection()
	_, err := inputColl.RemoveAll(nil)
	if err != nil {
		deps.Env.Error(err, nil)
		t.FailNow()
	}
	inputColl.Database.Session.Close()
}

func resetOutputColl(t *testing.T, deps *Dependencies) {
	outputColl, err := deps.Env.DB.NewOutputConnection("")
	if err != nil {
		deps.Env.Error(err, nil)
		t.FailNow()
	}
	_, err = outputColl.RemoveAll(nil)
	if err != nil {
		deps.Env.Error(err, nil)
		t.FailNow()
	}
	outputColl.Database.Session.Close()
}

func resetMetaDBDatabasesColl(t *testing.T, deps *Dependencies) {
	dbsColl := deps.Env.DB.NewMetaDBDatabasesConnection()
	_, err := dbsColl.RemoveAll(nil)
	if err != nil {
		deps.Env.Error(err, nil)
		t.FailNow()
	}
	dbsColl.Database.Session.Close()
}
