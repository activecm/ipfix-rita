package integrationtest

import (
	"flag"
	"fmt"
	"testing"
)

//TestFixture defines functions which update data referenced by Key
//at different points in the testing lifecycle.
//
//A test fixture may require other fixtures be available
//by placing the required fixture's key in the Requires field.
//No sorting is performed. Registration order matters.
//
//Each function may offer a new fixture to be stored as the first
//returned value. If the second returned value is false, the
//first return value will be ignored and no update will occur.
//Assignments to the passed in map are inneffectual as the map
//is a copy.
type TestFixture struct {
	Key           string
	Requires      []string
	LongRunning   bool //Skip this test fixture if testing.Short()
	BeforePackage func(FixtureData) (interface{}, bool)
	Before        func(*testing.T, FixtureData) (interface{}, bool)
	After         func(*testing.T, FixtureData) (interface{}, bool)
	AfterPackage  func(FixtureData) (interface{}, bool)
}

//FixtureData is used to access the data produced by registered TestFixtures
type FixtureData struct {
	interTestState  map[string]interface{}
	skippedFixtures map[string]bool
}

//Get returns the data produced by a TestFixture with a given key
func (f FixtureData) Get(key string) interface{} {
	return f.interTestState[key]
}

//GetWithSkip returns the data produced by a TestFixture with a given key.
//GetWithSkip will skip a test if the TestFixture identified by the given key
//is LongRunning.
func (f FixtureData) GetWithSkip(t *testing.T, key string) interface{} {
	if _, ok := f.skippedFixtures[key]; ok {
		t.Skip()
	}
	return f.interTestState[key]
}

//FixtureManager provides a managed way to provide
//inter-test state to tests under go test.
type FixtureManager struct {
	interTestState  map[string]interface{}
	skippedFixtures map[string]bool
	fixtures        []TestFixture
}

//NewFixtureManager instantiates a new FixtureManager
func NewFixtureManager() *FixtureManager {
	flag.Parse() //expect testing.Short() to work
	return &FixtureManager{
		interTestState:  make(map[string]interface{}),
		skippedFixtures: make(map[string]bool),
	}
}

//RegisterFixture adds a fixture to the FixtureManager.
//The BeforePackage and Before functions of each TestFixture
//run in the order that the fixtures were registered in.
//The AfterPackage and After functions of each TestFixture
//run in the opposite order that the fixtures were registered in.
func (d *FixtureManager) RegisterFixture(fixture TestFixture) {
	d.fixtures = append(d.fixtures, fixture)
	if fixture.LongRunning && testing.Short() {
		d.skippedFixtures[fixture.Key] = true
	}
}

//BeginTestPackage will call the BeforePackage function
//registered on each fixture
func (d *FixtureManager) BeginTestPackage() {
	for _, registeredFixture := range d.fixtures {
		_, shouldSkip := d.skippedFixtures[registeredFixture.Key]
		if registeredFixture.BeforePackage != nil && !shouldSkip {
			if registeredFixture.Requires != nil {
				for _, requiredFixture := range registeredFixture.Requires {
					if _, ok := d.interTestState[requiredFixture]; !ok {
						fmt.Printf(
							"error running %s for fixture %s. %s is required but it was not registered\n",
							"BeforePackage()", registeredFixture.Key, requiredFixture,
						)
					}
				}
			}
			newData, ok := registeredFixture.BeforePackage(d.getFixtureData())
			if ok {
				d.interTestState[registeredFixture.Key] = newData
			}
		}
	}
}

//BeginTest will call the Before function
//registered on each fixture
func (d *FixtureManager) BeginTest(t *testing.T) FixtureData {
	for _, registeredFixture := range d.fixtures {
		_, shouldSkip := d.skippedFixtures[registeredFixture.Key]
		if registeredFixture.Before != nil && !shouldSkip {
			if registeredFixture.Requires != nil {
				for _, requiredFixture := range registeredFixture.Requires {
					if _, ok := d.interTestState[requiredFixture]; !ok {
						fmt.Printf(
							"error running %s for fixture %s. %s is required but it was not registered\n",
							"Before()", registeredFixture.Key, requiredFixture,
						)
					}
				}
			}
			newData, ok := registeredFixture.Before(t, d.getFixtureData())
			if ok {
				d.interTestState[registeredFixture.Key] = newData
			}
		}
	}

	return d.getFixtureData()
}

//EndTest will call the After function
//registered on each fixture
func (d *FixtureManager) EndTest(t *testing.T) {
	for i := len(d.fixtures) - 1; i > -1; i-- {
		registeredFixture := d.fixtures[i]
		_, shouldSkip := d.skippedFixtures[registeredFixture.Key]
		if registeredFixture.After != nil && !shouldSkip {
			if registeredFixture.Requires != nil {
				for _, requiredFixture := range registeredFixture.Requires {
					if _, ok := d.interTestState[requiredFixture]; !ok {
						fmt.Printf(
							"error running %s for fixture %s. %s is required but it was not registered\n",
							"After()", registeredFixture.Key, requiredFixture,
						)
					}
				}
			}
			newData, ok := registeredFixture.After(t, d.getFixtureData())
			if ok {
				d.interTestState[registeredFixture.Key] = newData
			}
		}
	}
}

//EndTestPackage will call the AfterPackage function
//registered on each fixture
func (d *FixtureManager) EndTestPackage() {

	for i := len(d.fixtures) - 1; i > -1; i-- {
		registeredFixture := d.fixtures[i]
		_, shouldSkip := d.skippedFixtures[registeredFixture.Key]
		if registeredFixture.AfterPackage != nil && !shouldSkip {
			if registeredFixture.Requires != nil {
				for _, requiredFixture := range registeredFixture.Requires {
					if _, ok := d.interTestState[requiredFixture]; !ok {
						fmt.Printf(
							"error running %s for fixture %s. %s is required but it was not registered\n",
							"AfterPackage()", registeredFixture.Key, requiredFixture,
						)
					}
				}
			}
			newData, ok := registeredFixture.AfterPackage(d.getFixtureData())
			if ok {
				d.interTestState[registeredFixture.Key] = newData
			}
		}
	}
}

func (d *FixtureManager) getFixtureData() FixtureData {
	return FixtureData{
		d.interTestState,
		d.skippedFixtures,
	}
}
