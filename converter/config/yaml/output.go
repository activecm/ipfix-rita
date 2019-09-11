package yaml

import "github.com/activecm/ipfix-rita/converter/config"

//output implements config.Output
type output struct {
	RITAMongoDB ritaMongoDB `yaml:"RITA-MongoDB"`
}

func (o *output) GetRITAConfig() config.RITA {
	return &o.RITAMongoDB
}

//ritaMongoDB implements config.RITA
type ritaMongoDB struct {
	MongoDB mongoDBConnection `yaml:"MongoDB-Connection"`
	DBRoot  string            `yaml:"DBRoot"`
	MetaDB  string            `yaml:"MetaDB"`
	Strobe  strobe            `yaml:"Strobe"`
}

func (r *ritaMongoDB) GetConnectionConfig() config.MongoDBConnection {
	return &r.MongoDB
}

func (r *ritaMongoDB) GetDBRoot() string {
	return r.DBRoot
}

func (r *ritaMongoDB) GetMetaDB() string {
	return r.MetaDB
}

func (r *ritaMongoDB) GetStrobe() config.Strobe {
	return &r.Strobe
}

//strobe implements config.Strobe
type strobe struct {
	ConnectionLimit int `yaml:"ConnectionLimit"`
}

func (s *strobe) GetConnectionLimit() int {
	return s.ConnectionLimit
}
