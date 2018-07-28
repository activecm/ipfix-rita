package yaml

import "github.com/activecm/ipfix-rita/converter/config"

//input implements config.Input
type input struct {
	LogstashMongoDB logstashMongoDB `yaml:"Logstash-MongoDB"`
}

func (i *input) GetLogstashMongoDBConfig() config.LogstashMongoDB {
	return &i.LogstashMongoDB
}

//logstashMongoDB implements config.LogstashMongoDB
type logstashMongoDB struct {
	MongoDB    mongoDBConnection `yaml:"MongoDB-Connection"`
	Database   string            `yaml:"Database"`
	Collection string            `yaml:"Collection"`
}

func (l *logstashMongoDB) GetConnectionConfig() config.MongoDBConnection {
	return &l.MongoDB
}

func (l *logstashMongoDB) GetDatabase() string {
	return l.Database
}

func (l *logstashMongoDB) GetCollection() string {
	return l.Collection
}
