package environment

import (
	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/ipfix-rita/converter/database"
	"github.com/activecm/ipfix-rita/converter/logging"
)

//Environment is used to embed the methods provided by
//the logger, config manager, etc. into a given struct
//This alleviates passing around a method context/ resource bundle.
type Environment struct {
	config.Config
	logging.Logger
	database.DB
}
