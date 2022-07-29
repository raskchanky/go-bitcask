package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/raskchanky/go-bitcask"
)

type listCommand struct {
	ui  cli.Ui
	dir string
	key string
}

func (c *listCommand) Help() string {
	helpText := `
Usage: bitcask list <directory>

  List all of the keys in the database specified by <directory>.
  If <directory> doesn't exist, an error will be returned.

     $ bitcask list /opt/mydb
`

	return strings.TrimSpace(helpText)
}

func (c *listCommand) Synopsis() string {
	return "List all of the keys in an existing bitcask database."
}

func (c *listCommand) usage() string {
	return "Usage: bitcask list <directory>"
}

func (c *listCommand) Run(args []string) int {
	if len(args) != 1 {
		c.ui.Error(c.usage())
		return 1
	}

	db, err := bitcask.Open(args[0])
	if err != nil {
		c.ui.Error(fmt.Sprintf("Error: %v", err))
		return 2
	}
	defer func() { _ = db.Close() }()

	keys := db.List()

	for _, k := range keys {
		c.ui.Output(k)
	}

	return 0
}

func newListCommand() *listCommand {
	ui := &cli.BasicUi{
		Reader:      bufio.NewReader(os.Stdin),
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	return &listCommand{ui: ui}
}
