package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/raskchanky/go-bitcask"
)

type deleteCommand struct {
	ui  cli.Ui
	dir string
	key string
}

func (c *deleteCommand) Help() string {
	helpText := `
Usage: bitcask delete <directory> <key>

  Delete the key and value from the database specified by <directory>.
  If <directory> doesn't exist, an error will be returned.

     $ bitcask delete /opt/mydb foo
`

	return strings.TrimSpace(helpText)
}

func (c *deleteCommand) Synopsis() string {
	return "Delete the key and value from an existing bitcask database."
}

func (c *deleteCommand) usage() string {
	return "Usage: bitcask delete <directory> <key>"
}

func (c *deleteCommand) Run(args []string) int {
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

	err = db.Delete(args[1])
	if err != nil {
		c.ui.Error(fmt.Sprintf("Error: %v", err))
		return 3
	}

	return 0
}

func newDeleteCommand() *deleteCommand {
	ui := &cli.BasicUi{
		Reader:      bufio.NewReader(os.Stdin),
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	return &deleteCommand{ui: ui}
}
