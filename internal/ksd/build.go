package ksd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// The default name of the output directory
const OutDir = "kout"

func IsKustoSourceFile(ext string) bool {
	return ext == ".kql" || ext == ".csl" || ext == ".kusto"
}

type declaration struct {
	// name of table or function
	name string
	// signature of table or function
	signature string
	// body
	body string
	// the type of declaration
	declType declType
	// doc
	doc string

	// only used while parsing
	parseState int
}

type declType uint8

const (
	functionType = iota
	tableType
)

// Walks Kusto source files under srcRoot, and building the result files
// under outRoot.
//
// For each file, a corresponding file is built relative to outRoot.
// A Kusto source file is a file has any of the following file extensions:
// - .kql
// - .csl
// - .kusto
func Build(srcRoot string, outRoot string) error {
	srcRoot = filepath.Clean(srcRoot)
	outRoot = filepath.Clean(outRoot)
	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// skip any file in specified outRoot
		if strings.HasPrefix(path, outRoot) {
			return nil
		}

		if d.IsDir() {
			// skip any out directories
			if d.Name() == OutDir {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)
		if !IsKustoSourceFile(ext) {
			return nil
		}

		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			panic(fmt.Sprintf("calculating rel path of '%s' from root '%s: %v", path, outRoot, err))
		}
		outFile := filepath.Join(outRoot, rel)
		outDir := filepath.Dir(outFile)
		err = os.MkdirAll(outDir, 0777)
		if err != nil {
			return fmt.Errorf("creating outDir: %w", err)
		}
		reader, err := os.Open(path)
		if err != nil {
			return err
		}
		defer reader.Close()

		decl, err := parse(reader)
		if err != nil {
			return fmt.Errorf("parsing file %s: %w", rel, err)
		}

		writer, err := os.Create(outFile)
		if err != nil {
			return fmt.Errorf("creating out file: %w", err)
		}
		defer func() error {
			if err := writer.Close(); err != nil {
				return fmt.Errorf("writing out file: %w", err)
			}
			return nil
		}()

		err = write(writer, decl, filepath.Dir(rel))
		if err != nil {
			return fmt.Errorf("writing out file %s: %w", rel, err)
		}

		return nil
	})
}

func write(
	writer io.Writer,
	decl *declaration,
	folder string) error {
	// ensure all folders are forward slashes
	folder = strings.ReplaceAll(folder, "\\", "/")
	var err error
	switch decl.declType {
	case functionType:
		_, err = writer.Write([]byte(
			fmt.Sprintf(
				".create-or-alter function with (folder=\"%s\",docstring=\"%s\") %s%s ",
				folder, decl.doc, decl.name, decl.signature)))
	case tableType:
		_, err = writer.Write([]byte(
			fmt.Sprintf(
				".create-merge table %s%s\n",
				decl.name, decl.signature)))
	default:
		panic(fmt.Sprintf("unhandled declarationType: %d", decl.declType))
	}

	if err != nil {
		return err
	}

	_, err = writer.Write([]byte(decl.body))
	if err != nil {
		return err
	}

	return nil
}
