package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/cli"
)

var (
	Version   = "0.0.1"
	GitCommit string
)

type versionCommand struct {
	ui        cli.Ui
	version   string
	gitCommit string
}

func (v *versionCommand) Synopsis() string {
	return "Prints the bitcask version"
}

func (v *versionCommand) Help() string {
	helpText := `
Usage: bitcask version

  Prints the version for this bitcask CLI.

      $ bitcask version

  There are no arguments or flags to this command. Any additional arguments or
  flags are ignored.
`

	return strings.TrimSpace(helpText)
}

func (v *versionCommand) Run(_ []string) int {
	v.ui.Output(versionString())
	return 0
}

func newVersionCommand() *versionCommand {
	ui := &cli.BasicUi{
		Reader:      bufio.NewReader(os.Stdin),
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	return &versionCommand{ui: ui}
}

func versionString() string {
	return fmt.Sprintf("bitcask v%s ('%s')", Version, GitCommit)
}
