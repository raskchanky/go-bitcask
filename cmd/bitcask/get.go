package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/raskchanky/go-bitcask"
)

type getCommand struct {
	ui  cli.Ui
	dir string
	key string
}

func (c *getCommand) Help() string {
	helpText := `
Usage: bitcask get <directory> <key>

  Get the value for a given key from the database specified by <directory>.
  If <directory> doesn't exist, an error will be returned.

     $ bitcask get /opt/mydb foo
`

	return strings.TrimSpace(helpText)
}

func (c *getCommand) Synopsis() string {
	return "Get the value for a key from an existing bitcask database."
}

func (c *getCommand) usage() string {
	return "Usage: bitcask get <directory> <key>"
}

func (c *getCommand) Run(args []string) int {
	if len(args) != 2 {
		c.ui.Error(c.usage())
		return 1
	}

	db, err := bitcask.Open(args[0])
	if err != nil {
		c.ui.Error(fmt.Sprintf("Error: %v", err))
		return 2
	}
	defer func() { _ = db.Close() }()

	val, err := db.Get(args[1])
	if err != nil {
		c.ui.Error(fmt.Sprintf("Error: %v", err))
		return 3
	}

	c.ui.Output(string(val))

	return 0
}

func newGetCommand() *getCommand {
	ui := &cli.BasicUi{
		Reader:      bufio.NewReader(os.Stdin),
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	return &getCommand{ui: ui}
}
