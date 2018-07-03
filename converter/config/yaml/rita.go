package yaml

//rita implements config.RITA
type rita struct {
	DBRoot string `yaml:"DBRoot"`
	MetaDB string `yaml:"MetaDB"`
}

func (r *rita) GetDBRoot() string {
	return r.DBRoot
}

func (r *rita) GetMetaDB() string {
	return r.MetaDB
}
