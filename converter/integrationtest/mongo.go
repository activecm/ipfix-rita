package integrationtest

import (
	"context"

	"github.com/activecm/dbtest"
	"github.com/activecm/dbtest/docker"
)

//NewMongoDBContainerFixture returns a TestFixture
//which creates a dbtest.MongoDBContainer with the given key.
func NewMongoDBContainerFixture(key string) TestFixture {
	return TestFixture{
		Key:         key,
		LongRunning: true,
		Requires: []string{
			DockerLoaderFixture.Key,
		},
		BeforePackage: func(fixtures FixtureData) (interface{}, bool) {
			loader := fixtures.Get(DockerLoaderFixture.Key).(docker.Loader)
			mongoContainer, err := dbtest.NewMongoDBContainer(context.Background(), loader)
			if err != nil {
				panic(err)
			}
			return mongoContainer, true
		},
		AfterPackage: func(fixtures FixtureData) (interface{}, bool) {
			loader := fixtures.Get(DockerLoaderFixture.Key).(docker.Loader)
			mongoContainer := fixtures.Get(key).(dbtest.MongoDBContainer)
			err := loader.StopService(context.Background(), mongoContainer)
			if err != nil {
				panic(err)
			}
			return nil, true
		},
	}
}
