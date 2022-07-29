package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/raskchanky/go-bitcask"
)

type putCommand struct {
	ui    cli.Ui
	dir   string
	key   string
	value []byte
}

func (c *putCommand) Help() string {
	helpText := `
Usage: bitcask put <directory> <key> <value>

  Put a new key/value pair into the database specified by <directory>.
  If <directory> doesn't exist, it will be created.

     $ bitcask put /opt/mydb foo bar
`

	return strings.TrimSpace(helpText)
}

func (c *putCommand) Synopsis() string {
	return "Put a new key/value pair into a bitcask database."
}

func (c *putCommand) usage() string {
	return "Usage: bitcask put <directory> <key> <value>"
}

func (c *putCommand) Run(args []string) int {
	if len(args) != 3 {
		c.ui.Error(c.usage())
		return 1
	}

	db, err := bitcask.Open(args[0])
	if err != nil {
		c.ui.Error(fmt.Sprintf("Error: %v", err))
		return 2
	}
	defer func() { _ = db.Close() }()

	err = db.Put(args[1], []byte(args[2]))
	if err != nil {
		c.ui.Error(fmt.Sprintf("Error: %v", err))
		return 3
	}

	return 0
}

func newPutCommand() *putCommand {
	ui := &cli.BasicUi{
		Reader:      bufio.NewReader(os.Stdin),
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	return &putCommand{ui: ui}
}
