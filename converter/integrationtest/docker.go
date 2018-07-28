package integrationtest

import (
	"github.com/activecm/dbtest/docker"
)

//DockerLoaderFixture ensures a dbtest Docker loader
//is placed in the environment
var DockerLoaderFixture = TestFixture{
	Key:         "docker-loader",
	LongRunning: true,
	BeforePackage: func(FixtureData) (interface{}, bool) {
		loader, err := docker.NewLoader()
		if err != nil {
			panic(err)
		}
		return loader, true
	},
	AfterPackage: func(fixtures FixtureData) (interface{}, bool) {
		loader := fixtures.Get("docker-loader").(docker.Loader)
		err := loader.Close()
		if err != nil {
			panic(err)
		}
		//Nil out the fixture so no one uses the closed loader
		return nil, true
	},
}
