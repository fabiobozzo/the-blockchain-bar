package main

import (
	"fmt"
	"os"
)

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
