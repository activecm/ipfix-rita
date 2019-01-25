package flowmap

import "github.com/globalsign/mgo/bson"
import "github.com/pkg/errors"

var (
	//ErrDoesNotExist notes that there is not a field with the given name in the FlowMap
	ErrDoesNotExist = errors.New("field does not exist in FlowMap")
	//ErrTypeMismatch notes that there is not a field with the given name and type in the FlowMap
	ErrTypeMismatch = errors.New("value returned for given field does not match requested type")
)

//FlowMap provides convenience methods for safely accessing
//data in map[string]interface objects.
type FlowMap struct {
	innerMap stringMap
}

//NewFlowMapFromBSON adapts a bson.M object to a FlowMap
func NewFlowMapFromBSON(bsonMap bson.M) FlowMap {
	return FlowMap{
		innerMap: bsonStringMap(bsonMap),
	}
}

//NewFlowMap adapts a map[string]interface{} to a FlowMap
func NewFlowMap(data map[string]interface{}) FlowMap {
	return FlowMap{
		innerMap: mapStringMap(data),
	}
}

func (f FlowMap) get(field string) (interface{}, error) {
	val, ok := f.innerMap.Get(field)
	if !ok {
		return nil, errors.Wrapf(ErrDoesNotExist, "Field: %s", field)
	}
	return val, nil
}

//GetFlowMap returns a FlowMap backed by the inner map
//held in the given field
func (f FlowMap) GetFlowMap(field string) (FlowMap, error) {
	dataIface, err := f.get(field)
	if err != nil {
		return FlowMap{}, err
	}

	//Right now we only support bson.M and map[string]interface{}
	//for backing FlowMaps. Try both.
	rawMap, ok := dataIface.(map[string]interface{})
	if ok {
		return NewFlowMap(rawMap), nil
	}

	bsonMap, ok := dataIface.(bson.M)
	if ok {
		return NewFlowMapFromBSON(bsonMap), nil
	}

	return FlowMap{}, errors.Wrapf(
		ErrTypeMismatch,
		"Field: %s; Type: %s; Value: %+v",
		field,
		"map[string]interface{}|bson.M",
		dataIface,
	)
}

//GetAsInt returns the int value for a given field.
//If the data matching the field is not of type int, a cast may be performed.
//If the field does not exist or the data matching the field is not the
//appropriate type, an error is returned.
func (f FlowMap) GetAsInt(field string) (int, error) {
	valIface, err := f.get(field)
	if err != nil {
		return 0, err
	}

	switch val := valIface.(type) {
	case int32:
		return int(val), nil
	case int:
		return val, nil
	case int64:
		return int(val), nil
	case float32:
		return int(val), nil
	case float64:
		return int(val), nil
	default:
		return 0, errors.Wrapf(
			ErrTypeMismatch,
			"Field: %s; Type: %s; Value: %+v",
			field,
			"int32|int|int64|float32|float64",
			valIface,
		)
	}
}

//GetAsInt64 returns the int64 value for a given field.
//If the data matching the field is not of type int64, a cast may be performed.
//If the field does not exist or the data matching the field is not the
//appropriate type, an error is returned.
func (f FlowMap) GetAsInt64(field string) (int64, error) {
	valIface, err := f.get(field)
	if err != nil {
		return 0, err
	}

	switch val := valIface.(type) {
	case int32:
		return int64(val), nil
	case int:
		return int64(val), nil
	case int64:
		return val, nil
	case float32:
		return int64(val), nil
	case float64:
		return int64(val), nil
	default:
		return 0, errors.Wrapf(
			ErrTypeMismatch,
			"Field: %s; Type: %s; Value: %+v",
			field,
			"int32|int|int64|float32|float64",
			valIface,
		)
	}
}

//GetString returns the string value for a given field.
//If the field does not exist or the data matching the field is not the
//appropriate type, an error is returned.
func (f FlowMap) GetString(field string) (string, error) {
	val, err := f.get(field)
	if err != nil {
		return "", err
	}

	strVal, ok := val.(string)
	if ok {
		return strVal, nil
	}
	return "", errors.Wrapf(
		ErrTypeMismatch,
		"Field: %s; Type: %s; Value: %+v",
		field,
		"string",
		val,
	)
}

//GetObjectID returns the bson.ObjectId value for a given field.
//If the field does not exist or the data matching the field is not the
//appropriate type, an error is returned.
func (f FlowMap) GetObjectID(field string) (bson.ObjectId, error) {
	val, err := f.get(field)
	if err != nil {
		return "", err
	}

	idVal, ok := val.(bson.ObjectId)
	if ok {
		return idVal, nil
	}
	return "", errors.Wrapf(
		ErrTypeMismatch,
		"Field: %s; Type: %s; Value: %+v",
		field,
		"bson.ObjectId",
		val,
	)
}
