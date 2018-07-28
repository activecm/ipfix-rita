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
