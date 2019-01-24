package safemap

import "gopkg.in/mgo.v2/bson"
import "github.com/pkg/errors"

var (
	//ErrDoesNotExist notes that there is not a field with the given name in the SafeMap
	ErrDoesNotExist = errors.New("field does not exist in SafeMap")
	//ErrTypeMismatch notes that there is not a field with the given name and type in the SafeMap
	ErrTypeMismatch = errors.New("value returned for given field does not match requested type")
)

//SafeMap provides convenience methods for safely accessing
//data in map[string]interface objects.
type SafeMap struct {
	innerMap stringMap
}

//NewSafeMapFromBSON adapts a bson.M object to a SafeMap
func NewSafeMapFromBSON(bsonMap bson.M) SafeMap {
	return SafeMap{
		innerMap: bsonStringMap(bsonMap),
	}
}

//NewSafeMap adapts a map[string]interface{} to a SafeMap
func NewSafeMap(data map[string]interface{}) SafeMap {
	return SafeMap{
		innerMap: mapStringMap(data),
	}
}

func (u SafeMap) get(field string) (interface{}, error) {
	val, ok := u.innerMap.Get(field)
	if !ok {
		return nil, ErrDoesNotExist
	}
	return val, nil
}

//GetSafeMap returns a SafeMap backed by the inner map
//held in the given field
func (u SafeMap) GetSafeMap(field string) (SafeMap, error) {
	dataIface, err := u.get(field)
	if err != nil {
		return SafeMap{}, errors.Wrap(err, "Field: "+field)
	}

	//Right now we only support bson.M and map[string]interface{}
	//for backing SafeMaps. Try both.
	rawMap, ok := dataIface.(map[string]interface{})
	if ok {
		return NewSafeMap(rawMap), nil
	}

	bsonMap, ok := dataIface.(bson.M)
	if ok {
		return NewSafeMapFromBSON(bsonMap), nil
	}

	return SafeMap{}, errors.Wrap(
		ErrTypeMismatch,
		"Field: "+field+"; Type: map[string]interface{}/ bson.M",
	)
}

//GetInt returns the int value for a given field.
//If the field does not exist or the data matching the field is not the
//appropriate type, an error is returned.
func (u SafeMap) GetInt(field string) (int, error) {
	val, err := u.get(field)
	if err != nil {
		return 0, errors.Wrap(err, "Field: "+field)
	}

	intVal, ok := val.(int)
	if ok {
		return intVal, nil
	}
	return 0, errors.Wrap(
		ErrTypeMismatch,
		"Field: "+field+"; Type: int",
	)
}

//GetIntAsInt64 returns the int64 value for a given field.
//If the data matching the field is of type int, a cast will be performed.
//If the field does not exist or the data matching the field is not the
//appropriate type, an error is returned.
func (u SafeMap) GetIntAsInt64(field string) (int64, error) {
	val, err := u.get(field)
	if err != nil {
		return 0, errors.Wrap(err, "Field: "+field)
	}

	int64Val, ok := val.(int64)
	if ok {
		return int64Val, nil
	}

	intVal, ok := val.(int)
	if ok {
		return int64(intVal), nil
	}
	return 0, errors.Wrap(
		ErrTypeMismatch,
		"Field: "+field+"; Type: int64/ int",
	)
}

//GetString returns the string value for a given field.
//If the field does not exist or the data matching the field is not the
//appropriate type, an error is returned.
func (u SafeMap) GetString(field string) (string, error) {
	val, err := u.get(field)
	if err != nil {
		return "", errors.Wrap(err, "Field: "+field)
	}

	strVal, ok := val.(string)
	if ok {
		return strVal, nil
	}
	return "", errors.Wrap(
		ErrTypeMismatch,
		"Field: "+field+"; Type: string",
	)
}

//GetObjectID returns the bson.ObjectId value for a given field.
//If the field does not exist or the data matching the field is not the
//appropriate type, an error is returned.
func (u SafeMap) GetObjectID(field string) (bson.ObjectId, error) {
	val, err := u.get(field)
	if err != nil {
		return "", errors.Wrap(err, "Field: "+field)
	}

	idVal, ok := val.(bson.ObjectId)
	if ok {
		return idVal, nil
	}
	return "", errors.Wrap(
		ErrTypeMismatch,
		"Field: "+field+"; Type: bson.ObjectId",
	)
}
