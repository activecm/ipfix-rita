package yaml

//rita implements config.RITA
type rita struct {
	DBRoot string `yaml:"DBRoot"`
}

func (r *rita) GetDBRoot() string {
	return r.DBRoot
}
