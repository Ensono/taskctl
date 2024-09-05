package main

import (
	"context"
	"os"

	cmd "github.com/Ensono/taskctl/cmd/taskctl"
)

func subCommands() (commandNames []string) {
	for _, command := range cmd.TaskCtlCmd.Commands() {
		commandNames = append(commandNames, append(command.Aliases, command.Name())...)
	}
	return
}

func setDefaultCommandIfNonePresent() {
	if len(os.Args) > 1 {
		// This will turn `taskctl [pipeline task]` => `taskctl run [pipeline task]`
		potentialCommand := os.Args[1]
		for _, command := range subCommands() {
			if command == potentialCommand {
				return
			}
		}
		os.Args = append([]string{os.Args[0], "run"}, os.Args[1:]...)
	}
}

func main() {
	// This is only here for backwards compatibility
	//
	// If any user names a runnable task or pipeline the same as
	// an existing command command will always take precedence ;)
	// And will most likely fail as the argument into the command was perceived as a command name
	setDefaultCommandIfNonePresent()
	// init loggerHere or in init function
	cmd.Execute(context.Background())
}
