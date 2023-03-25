package ksd

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type declaration struct {
	// name of table or function
	name string
	// signature of table or function
	signature string
	// body
	body string
	// the type of declaration
	declType declType

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
	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".kql" && ext != ".csl" && ext != ".kusto" {
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

		err = build(reader, writer, filepath.Dir(rel))
		if err != nil {
			return fmt.Errorf("building file %s: %w", rel, err)
		}

		return nil
	})
}

func build(reader io.Reader, writer io.Writer, folder string) error {
	scanner := bufio.NewScanner(reader)
	letFound := false
	comments := make([]string, 0, 20)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "let") {
			letFound = true
			break
		}

		comments = append(comments, line)
	}

	if !letFound {
		return fmt.Errorf("missing 'let' statement")
	}

	start := len(comments) - 1
	for start >= 0 && strings.HasPrefix(comments[start], "//") {
		start--
	}

	var docs strings.Builder
	var docstring string
	if start < len(comments)-1 {
		for i := start + 1; i < len(comments); i++ {
			comment := strings.TrimSpace(comments[i][2:])
			// escape all double-quotes
			comment = strings.ReplaceAll(comment, "\"", "\\\"")
			docs.WriteString(comment)
			docs.WriteString(" ")
		}
		docstring = docs.String()[:docs.Len()-1]
	}

	// first scan leftover words from Text()
	lineScanner := bufio.NewScanner(strings.NewReader(scanner.Text()))
	lineScanner.Split(bufio.ScanWords)
	decl := &declaration{}
	// parse from remaining line
	err := parseDeclaration(lineScanner, reader, decl)
	if err != nil {
		return fmt.Errorf("parsing declaration: %w", err)
	}

	// parse from remaining text
	scanner = bufio.NewScanner(reader)
	scanner.Split(bufio.ScanWords)
	err = parseDeclaration(scanner, reader, decl)
	if err != nil {
		return fmt.Errorf("parsing declaration: %w", err)
	}

	// write the transpiled version
	err = write(writer, docstring, decl, folder)
	if err != nil {
		return fmt.Errorf("writing out file: %w", err)
	}
	return nil
}

func parseDeclaration(
	scanner *bufio.Scanner,
	underlyingReader io.Reader,
	decl *declaration) error {
	// We parse two kinds of declaration:
	// functions:
	// let \s+ {name} \s+ = \s+ ({signature}) \s+ {
	// tables:
	// let \s+ {name} \s+ = \s+ datatable \s+ ({signature}) \s+ {
	for scanner.Scan() {
		switch decl.parseState {
		case 0: // let
			if scanner.Text() != "let" {
				return fmt.Errorf("let statement not found in '%s'", scanner.Text())
			}
		case 1: // name
			decl.name = scanner.Text()
		case 2: // '='
			if scanner.Text() != "=" {
				return fmt.Errorf("expected '=' after 'let' statement, i.e. let myFunc = (arg1:string){}")
			}
		case 3: // [datatable] ({signature})
			if scanner.Text() == "datatable" {
				decl.declType = tableType
				// now, scan to next "(" token so datatable and functions
				// are at the same location
				scanner.Scan()
				if scanner.Err() != nil {
					return scanner.Err()
				}
			}

			if !strings.HasPrefix(scanner.Text(), "(") {
				switch decl.declType {
				case functionType:
					return fmt.Errorf("expected '(' after '=' in let statement, i.e. let myFunc = (arg1:string){}")
				case tableType:
					return fmt.Errorf("expected '(' after 'datatable' in let statement, i.e. let myTable = datatable (['foo']:string) []")
				}
			}

			decl.signature = scanner.Text()
			log.Printf("signature start: %s", decl.signature)
			// scan until next ')' character
			pscanner := bufio.NewScanner(underlyingReader)
			pscanner.Split(scanParenthesisEnd)
			if !pscanner.Scan() {
				if pscanner.Err() != nil {
					return pscanner.Err()
				}
				return fmt.Errorf("unclosed parenthesis, closing ')' not found in let statement")
			}
			if pscanner.Err() != nil {
				return pscanner.Err()
			}

			decl.signature += pscanner.Text()
			log.Printf("signature end: %s", decl.signature)
		case 4: // for function '{<char', for table: '[ ]' and done
			switch decl.declType {
			case functionType:
				if !strings.HasPrefix(scanner.Text(), "{") {
					return fmt.Errorf("missing block declaration '{' after method signature in let statement, i.e. let myFunc = (arg1:string) { Table1 | limit 10 }")
				}
				decl.body = scanner.Text()
			case tableType:
				// for table, we will parse remaining text and return.
				if !strings.HasPrefix(scanner.Text(), "[") {
					return fmt.Errorf("missing block declaration '[' after method signature in let statement, i.e. let myTable = datatable (['foo']:string) []")
				}

				if strings.HasSuffix(scanner.Text(), "]") {
					// recommended, single "[]"
					return nil
				}

				wscanner := bufio.NewScanner(underlyingReader)
				wscanner.Split(scanParenthesisEnd)
				has := wscanner.Scan()
				if !has {
					return fmt.Errorf("unclosed brackets, closing ']' not found in body declaration")
				}

				if wscanner.Text() == "]" {
					// recommended, "[ ]"
					return nil
				} else if !strings.HasSuffix(wscanner.Text(), "]") {
					return fmt.Errorf("unclosed brackets, closing ']' not found in body declaration")
				}

				fmt.Printf("WARNING: Syncing data within datatable syntax is not currently supported. The following contents will be ignored:\n%s", scanner.Text()[:len(scanner.Text())-1])
				return nil
			default:
				panic(fmt.Sprintf("unsupported decl type: %d", decl.declType))
			}
			scanner = bufio.NewScanner(underlyingReader)
			scanner.Split(bufio.ScanLines)
		default:
			switch decl.declType {
			case functionType:
				var sb strings.Builder
				sb.WriteString(decl.body)
				sb.WriteString("\n")
				sb.WriteString(scanner.Text())

				for scanner.Scan() {
					sb.WriteString(scanner.Text())
					sb.WriteString("\n")
				}
				decl.body = sb.String()
			}

			return scanner.Err()
		}
		decl.parseState += 1
	}

	return scanner.Err()
}

func write(
	writer io.Writer,
	doc string,
	decl *declaration,
	folder string) error {
	var err error
	switch decl.declType {
	case functionType:
		_, err = writer.Write([]byte(
			fmt.Sprintf(
				".create-or-alter function with (folder=\"%s\",docstring=\"%s\") %s%s\n",
				folder, doc, decl.name, decl.signature)))
	case tableType:
		_, err = writer.Write([]byte(
			fmt.Sprintf(
				".create-merge table %s%s (folder=\"%s\",docstring=\"%s\")\n",
				decl.name, decl.signature, folder, doc)))
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

func scanParenthesisEnd(data []byte, atEOF bool) (advance int, token []byte, err error) {
	start := 0
	for ; start < len(data); start++ {
		if data[start] == ')' {
			return start + 1, data[:start+1], nil
		}
	}

	if atEOF {
		return len(data), nil, bufio.ErrFinalToken
	}

	// request more data
	return 0, nil, nil
}
