package main

import "os"

var (
	version = "0.1.0"
)

func main() {
	os.Exit(NewCli().Parse().Run())
}
