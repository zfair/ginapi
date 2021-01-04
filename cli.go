package main

import (
	"errors"
	"flag"
	"fmt"
)

var (
	ErrCliNoInpath = errors.New("expected input path")
	ErrCliBadMode  = errors.New("bad codegen mode, expected server/client")
)

type ginapiCli struct {
	*codegen

	version string

	isHelp    bool
	isVersion bool
	mode      string
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
	flag.StringVar(&c.mode, "m", CodegenModeServer, "server/client code to generate")

	flag.Parse()

	return c
}

func (c *ginapiCli) validate() error {
	if c.inpath == "" {
		return ErrCliNoInpath
	}
	if c.mode == "" {
		return ErrCliBadMode
	}
	if _, ok := CodegenModes[c.mode]; !ok {
		return fmt.Errorf("%w: %s", ErrCliBadMode, c.mode)
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

	if err := c.codegen.Run(); err != nil {
		fmt.Println("ERROR: codegen:", err)
		return 1
	}

	return
}
