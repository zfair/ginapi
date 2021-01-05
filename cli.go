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

type GinapiCli struct {
	*Codegen

	version string

	isHelp    bool
	isVersion bool
	mode      string
}

func NewCli() *GinapiCli {
	return &GinapiCli{
		version: version,
		Codegen: NewCodegen(),
	}
}

func (*GinapiCli) showUsage() {
	fmt.Print("ginapi is a dead simple OpenAPI codegen for Gin.\n\n")
	flag.PrintDefaults()
}

func (c *GinapiCli) showVersion() {
	fmt.Printf("ginapi v%s\n", c.version)
}

func (c *GinapiCli) Parse() *GinapiCli {
	flag.Usage = c.showUsage

	flag.BoolVar(&c.isHelp, "h", false, "show help")
	flag.BoolVar(&c.isVersion, "v", false, "show version")
	flag.StringVar(&c.inpath, "i", "", "path to OpenAPI generated code as input")
	flag.StringVar(&c.mode, "m", CodegenModeServer, "server/client code to generate")

	flag.Parse()

	return c
}

func (c *GinapiCli) validate() error {
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

func (c *GinapiCli) Run() (rc int) {
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

	if err := c.Codegen.Run(); err != nil {
		fmt.Println("ERROR: codegen:", err)
		return 1
	}

	return
}
