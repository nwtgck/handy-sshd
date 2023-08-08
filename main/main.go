package main

import (
	"fmt"
	"github.com/nwtgck/handy-sshd/cmd"
	"os"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(-1)
	}
}
