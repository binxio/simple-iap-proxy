package main

import (
	"fmt"
	"github.com/binxio/simple-iap-proxy/cmd"
	"os"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}
