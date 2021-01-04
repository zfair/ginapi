package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrCodegenInpathNotExists = errors.New("input path not exists")
	ErrCodegenInpathNotDir    = errors.New("input path not a directory")
)

type codegen struct {
	outpath string
}

func Codegen() *codegen {
	return &codegen{}
}

func (c *codegen) Run(inpath string) error {
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
