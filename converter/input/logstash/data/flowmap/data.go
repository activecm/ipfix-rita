package flowmap

import (
	"github.com/globalsign/mgo/bson"
)

//stringMap returns arbitrary data given a field name.
//stringMap is used to abstract over map[string]interface{}
//and bson.M via MapStringMap and BsonStringMap respectively.
type stringMap interface {
	//Get retreives arbitrary data for a given field name
	Get(field string) (interface{}, bool)
}

//mapStringMap adapts map[string]interface{} objects
//to StringMap objects
type mapStringMap map[string]interface{}

//Get retreives arbitrary data for a given field name
func (m mapStringMap) Get(field string) (interface{}, bool) {
	val, ok := m[field]
	return val, ok
}

//bsonStringMap adapts bson.M objects to StringMap objects
type bsonStringMap bson.M

//Get retreives arbitrary data for a given field name
func (b bsonStringMap) Get(field string) (interface{}, bool) {
	val, ok := b[field]
	return val, ok
}
