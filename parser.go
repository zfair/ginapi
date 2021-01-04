package main

import (
	"fmt"
	goparser "go/parser"
	gotoken "go/token"
	"path/filepath"
)

type parser struct {
	inpath  string
	srcpath string
}

func Parser() *parser {
	return &parser{}
}

func (p *parser) Parse() error {
	p.srcpath = filepath.Join(p.inpath, "go")

	fset := gotoken.NewFileSet()
	pkgs, err := goparser.ParseDir(fset, p.srcpath, nil, goparser.ParseComments)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			fmt.Println(name)
			fmt.Println(file.Name)
			// TODO
		}
	}

	return nil
}
