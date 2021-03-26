package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
)

var (
	ErrCliNoInpath = errors.New("expected input path")
)

type GinapiCli struct {
	*Codegen

	version string

	isHelp    bool
	isVersion bool
	rawVars   string
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
	flag.StringVar(&c.rawVars, "vars", "", "server variables as JSON")
	flag.BoolVar(&c.isGinCtx, "ctx", false, "enable `*gin.Context` as an argument")

	flag.Parse()
	return c
}

func (c *GinapiCli) validate() error {
	if c.inpath == "" {
		return ErrCliNoInpath
	}
	if raw := c.rawVars; raw != "" {
		if err := json.Unmarshal([]byte(raw), &c.vars); err != nil {
			return err
		}
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
