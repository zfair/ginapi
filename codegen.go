package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	CodegenModeServer = "server"
	CodegenModeClient = "client"
)

var (
	CodegenModes = map[string]struct{}{
		CodegenModeServer: {},
		CodegenModeClient: {},
	}
)

var (
	ErrCodegenInpathNotExists = errors.New("input path not exists")
	ErrCodegenInpathNotDir    = errors.New("input path not a directory")
)

type Codegen struct {
	*Parser

	outpath string
	mode    string
}

func NewCodegen() *Codegen {
	return &Codegen{
		Parser: NewParser(),
	}
}

func (c *Codegen) MkOutpath() error {
	inpath := c.inpath

	info, err := os.Stat(inpath)
	if os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrCodegenInpathNotExists, inpath)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrCodegenInpathNotDir, inpath)
	}

	c.outpath = filepath.Join(inpath, "ginapi")
	_ = os.MkdirAll(c.outpath, os.ModePerm)

	return nil
}

func (c *Codegen) Run() error {
	if err := c.MkOutpath(); err != nil {
		return err
	}
	if err := c.Parser.Parse(); err != nil {
		return err
	}
	return nil
}
