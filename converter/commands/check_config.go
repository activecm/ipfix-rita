package commands

import (
	"fmt"

	"github.com/activecm/ipfix-rita/converter/config"
	"github.com/activecm/ipfix-rita/converter/config/yaml"
	"github.com/activecm/ipfix-rita/converter/input/logstash/mongodb"
	"github.com/activecm/ipfix-rita/converter/output/rita"
	"github.com/urfave/cli"
)

func init() {
	GetRegistry().RegisterCommands(cli.Command{
		Name:  "check-config",
		Usage: "Test the ipfix-rita configuration and test the MongoDB connection",
		Action: func(c *cli.Context) error {
			configBuff, err := yaml.ReadConfigFile()
			if err != nil {
				return cli.NewExitError(fmt.Sprintf("%+v\n", err), 1)
			}
			if len(configBuff) == 0 {
				return cli.NewExitError(fmt.Sprintf("empty config file"), 1)
			}

			conf, err := yaml.NewYAMLConfig(configBuff)
			if err != nil {
				return cli.NewExitError(fmt.Sprintf("%+v\n", err), 1)
			}

			serialConfig := conf.(config.Serializable)
			confStr, err := serialConfig.SaveConfig()
			if err != nil {
				return cli.NewExitError(fmt.Sprintf("%+v\n", err), 1)
			}
			fmt.Printf("Loaded Configuration:\n%s\n", confStr)

			db, err := mongodb.NewLogstashMongoInputDB(conf.GetInputConfig().GetLogstashMongoDBConfig())
			if err != nil {
				return cli.NewExitError(fmt.Sprintf("%+v\n", err), 1)
			}
			err = db.Ping()
			if err != nil {
				return cli.NewExitError(fmt.Sprintf("%+v\n", err), 1)
			}
			fmt.Printf("Input Database Connection Successful\n")

			coll := db.NewInputConnection()
			count, err := coll.Count()
			if err != nil {
				return cli.NewExitError(fmt.Sprintf("%+v\n", err), 1)
			}
			fmt.Printf("Found %d Flow Records Ready For Processing\n", count)
			coll.Database.Session.Close()

			outDB, err := rita.NewOutputDB(conf.GetOutputConfig().GetRITAConfig())
			if err != nil {
				return cli.NewExitError(fmt.Sprintf("%+v\n", err), 1)
			}
			err = outDB.Ping()
			if err != nil {
				return cli.NewExitError(fmt.Sprintf("%+v\n", err), 1)
			}
			fmt.Printf("Output Database Connection Successful\n")
			return nil
		},
	})
}
