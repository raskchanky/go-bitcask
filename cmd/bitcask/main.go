package main

import (
	"log"
	"os"

	"github.com/mitchellh/cli"
)

func main() {
	c := cli.NewCLI("bitcask", versionString())
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"version": func() (cli.Command, error) {
			return newVersionCommand(), nil
		},
		"put": func() (cli.Command, error) {
			return newPutCommand(), nil
		},
		"get": func() (cli.Command, error) {
			return newGetCommand(), nil
		},
		"list": func() (cli.Command, error) {
			return newListCommand(), nil
		},
		"delete": func() (cli.Command, error) {
			return newDeleteCommand(), nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}

	os.Exit(exitStatus)
}
