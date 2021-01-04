package main

import (
	"errors"
	"flag"
	"fmt"
)

var (
	ErrCliNoInpath = errors.New("expected input path")
)

type ginapiCli struct {
	version string

	isHelp    bool
	isVersion bool
	inpath    string

	codegen *codegen
}

func Cli() *ginapiCli {
	return &ginapiCli{
		version: version,
		codegen: Codegen(),
	}
}

func (*ginapiCli) showUsage() {
	fmt.Print("ginapi is a dead simple OpenAPI codegen for Gin.\n\n")
	flag.PrintDefaults()
}

func (c *ginapiCli) showVersion() {
	fmt.Printf("ginapi v%s\n", c.version)
}

func (c *ginapiCli) Parse() *ginapiCli {
	flag.Usage = c.showUsage

	flag.BoolVar(&c.isHelp, "h", false, "show help")
	flag.BoolVar(&c.isVersion, "v", false, "show version")
	flag.StringVar(&c.inpath, "i", "", "path to OpenAPI generated code as input")

	flag.Parse()

	return c
}

func (c *ginapiCli) validate() error {
	if c.inpath == "" {
		return ErrCliNoInpath
	}
	return nil
}

func (c *ginapiCli) Run() (rc int) {
	if c.isHelp {
		flag.Usage()
		return
	}

	if c.isVersion {
		c.showVersion()
		return
	}

	if err := c.validate(); err != nil {
		fmt.Println("ERROR: cli:", err)
		return 1
	}

	if err := c.codegen.Run(c.inpath); err != nil {
		fmt.Println("ERROR: codegen:", err)
		return 1
	}

	return
}
