package commands

import "github.com/urfave/cli"

//Registry provides a common way to make cli.Commands available
//to an application
type Registry struct {
	commands []cli.Command
}

//RegisterCommands makes adds commands to the Registry
func (r *Registry) RegisterCommands(command ...cli.Command) {
	r.commands = append(r.commands, command...)
}

//GetCommands returns all registered commands
func (r *Registry) GetCommands() []cli.Command {
	//return a new slice so callers can't muck with the _commands slice
	return r.commands[:]
}

//_registry provides a backing singleton for GetRegistry
var _registry *Registry

//GetRegistry returns a singleton instance of Registry
func GetRegistry() *Registry {
	if _registry == nil {
		_registry = &Registry{}
	}
	return _registry
}
