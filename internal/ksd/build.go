package ksd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"
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

type lexer struct {
	r        *bufio.Reader
	token    string
	tokenBuf strings.Builder

	err error
	row int64
	col int
}

func newLexer(reader *bufio.Reader) *lexer {
	return &lexer{
		r: reader,
	}
}

// advances the cursor to past the newline character.
//
// the line, including newline character, is saved to tokenBuf.
func (l *lexer) readLine() (hasMore bool) {
	line, err := l.r.ReadString('\n')
	if err == io.EOF {
		return false
	} else if err != nil {
		l.err = err
		return false
	}

	l.row += 1
	l.col = 0
	l.tokenBuf.WriteString(line)
	return true
}

// advances the cursor to past the newline character.
//
// the tokenBuf is consumed as token, and tokenBuf is reset.
func (l *lexer) consumeLine() (hasMore bool) {
	hasMore = l.readLine()
	l.token = l.tokenBuf.String()
	l.tokenBuf.Reset()
	return hasMore
}

// advances the cursor to the first non-space character.
//
// whitespace is discarded.
// the first non-space character is saved into the tokenBuf.
func (l *lexer) skipSpace() (hasMore bool) {
	for {
		r, _, err := l.r.ReadRune()
		if err == io.EOF {
			return false
		} else if err != nil {
			l.err = err
			return false
		}

		if r == '\n' {
			l.row += 1
			l.col = 1
		} else {
			l.col += 1
		}

		if !unicode.IsSpace(r) {
			_, err := l.tokenBuf.WriteRune(r)
			if err != nil {
				l.err = err
				return false
			}

			break
		}
	}

	return true
}

// advances the cursor to the first space character.
// the tokenBuf is consumed as token, and tokenBuf is reset.
func (l *lexer) consumeToken() (hasMore bool) {
	more := l.readToken()

	if l.tokenBuf.Len() > 0 {
		l.token = l.tokenBuf.String()
		l.tokenBuf.Reset()
	}

	return more
}

// advances the cursor to the first space character.
// the character is written to tokenBuf.
func (l *lexer) readToken() (hasMore bool) {
	for {
		r, _, err := l.r.ReadRune()
		if err == io.EOF {
			return false
		} else if err != nil {
			l.err = err
			return false
		}

		if r == '\n' {
			l.row += 1
			l.col = 1
		} else {
			l.col += 1
		}

		if unicode.IsSpace(r) {
			break
		}

		_, err = l.tokenBuf.WriteRune(r)
		if err != nil {
			l.err = err
			return false
		}
	}

	return true
}

// advances the cursor to one-past the first occurence of delim byte.
//
// the presence of the delim byte inside a quoted string,
// single-quote or double-quote, is considered an escape
// character and is ignored.
//
// all bytes read (including whitespace and delim) are saved to
// token, and the tokenBuf is cleared.
func (l *lexer) consumeTill(s byte) (hasMore bool) {
	inString := false
	for {
		r, _, err := l.r.ReadRune()
		if err == io.EOF {
			return false
		} else if err != nil {
			l.err = err
			return false
		}

		if r == '\n' {
			l.row += 1
			l.col = 1
		} else {
			l.col += 1
		}

		if r == '\'' || r == '"' {
			inString = !inString
		}

		_, err = l.tokenBuf.WriteRune(r)
		if err != nil {
			l.err = err
			return false
		}

		if !inString && byte(r) == s {
			break
		}
	}

	l.token = l.tokenBuf.String()
	l.tokenBuf.Reset()
	return true
}

var errNonDelimWhitespaceFound = errors.New("non-whitespace, non-delim char found")

// advances the cursor to one-past the first occurence of delim byte.
//
// whitespace is ignored and discarded when advancing the cursor.
// if non-whitespace, non-delim char is found, errNonDelimWhitespaceFound is returned.
//
// the delim byte is saved into token, the tokenBuf is reset.
func (l *lexer) consumeSpaced(s byte) (hasMore bool) {
	l.readSpaced(s)
	l.token = l.tokenBuf.String()
	l.tokenBuf.Reset()
	return true
}

// advances the cursor to one-past the first occurence of delim byte.
//
// whitespace is ignored and discarded when advancing the cursor.
// if non-whitespace, non-delim char is found, errNonDelimWhitespaceFound is returned.
//
// the delim byte is saved into tokenBuf.
func (l *lexer) readSpaced(s byte) (hasMore bool) {
	for {
		r, _, err := l.r.ReadRune()
		if err == io.EOF {
			return false
		} else if err != nil {
			l.err = err
			return false
		}

		if r == '\n' {
			l.row += 1
			l.col = 1
		} else {
			l.col += 1
		}

		if unicode.IsSpace(r) {
			continue
		} else if byte(r) == s {
			_, err = l.tokenBuf.WriteRune(r)
			if err != nil {
				l.err = err
				return false
			}
		}

		l.err = errNonDelimWhitespaceFound
		return false
	}
}

type ParseError struct {
	row int64
	col int
	msg string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("[%d,%d] %s", e.row, e.col, e.msg)
}

func (l *lexer) Errorf(m string, args ...any) error {
	return &ParseError{
		row: l.row,
		col: l.col,
		msg: fmt.Sprintf(m, args...),
	}
}

func build(reader io.ReadSeeker, writer io.Writer, folder string) error {
	lexer := newLexer(bufio.NewReader(reader))
	letFound := false
	comments := make([]string, 0, 20)
	for lexer.consumeLine() {
		line := lexer.token
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

	_, err := reader.Seek(lexer.row-1, io.SeekStart)
	if err != nil {
		return fmt.Errorf("seeking file: %w", err)
	}
	lexer.row -= 1
	lexer.col = 0
	lexer.r.Reset(reader)
	lexer.tokenBuf.Reset()
	lexer.consumeLine()

	decl := &declaration{}
	err = parseDeclaration(lexer, decl)
	if err != nil {
		return fmt.Errorf("parsing declaration: %w", err)
	}

	decl.body = lexer.tokenBuf.String()
	lexer.tokenBuf.Reset()

	// write the transpiled version
	err = write(writer, docstring, decl, folder)
	if err != nil {
		return fmt.Errorf("writing out file: %w", err)
	}
	return nil
}

func parseDeclaration(
	lex *lexer,
	decl *declaration) error {
	// We parse two kinds of declaration:
	// functions:
	// let \s+ {name} \s*  = \s* ({signature}) \s* {
	// tables:
	// let \s+ {name} \s*  = \s* datatable \s* ({signature}) \s+ {
	for {
		switch decl.parseState {
		case 0: // \s*let
			lex.skipSpace()
			lex.consumeToken()
			if lex.token != "let" {
				return lex.Errorf("expected 'let' statement, found %s", lex.token)
			}
		case 1: // \s+name
			lex.skipSpace()
			if !lex.consumeToken() {
				if lex.err != nil {
					return lex.err
				}
				return lex.Errorf("incomplete 'let' statement, expected identfier")
			}
			decl.name = lex.token
			log.Printf("name: %s", decl.name)
		case 2: // \s*'='
			lex.consumeSpaced('=')
			if lex.err != nil {
				return lex.err
			}

			if lex.token != "=" {
				return lex.Errorf("expected variable assignment '=', i.e. 'let %s ='", decl.name)
			}
		case 3: // \s*[datatable]\s*({signature})
			lex.consumeSpaced('d') // TODO: or '('
			lex.readToken()        // read, but don't consume
			if lex.err != nil {
				return lex.err
			}

			if lex.tokenBuf.String() == "datatable" {
				decl.declType = tableType
				lex.tokenBuf.Reset()
				// now, read to next '(' token
				lex.skipSpace()
				if lex.err != nil {
					return lex.err
				}
			}

			err := parseSignature(lex, decl)
			if err != nil {
				return err
			}
			log.Printf("signature: %s", decl.signature)
		case 4: // for function '{<body>', for table: '[ ]' and done
			lex.skipSpace()
			switch decl.declType {
			case functionType:
				lex.readToken()
				if lex.err != nil {
					return lex.err
				}
				if !strings.HasPrefix(lex.tokenBuf.String(), "{") {
					return lex.Errorf("missing block declaration '{' after method signature in let statement, i.e. let myFunc = (arg1:string) { Table1 | limit 10 }")
				}
			case tableType:
				lex.consumeToken()
				if lex.err != nil {
					return lex.err
				}
				// for table, we will parse remaining text and return.
				if !strings.HasPrefix(lex.token, "[") {
					return lex.Errorf("missing block declaration '[' after method signature in let statement, i.e. let myTable = datatable (['foo']:string) []")
				}

				if strings.HasSuffix(lex.token, "]") {
					// recommended, single "[]"
					return nil
				}

				// parsing something that looks like: '[<space>]'
				lex.consumeTill(']')
				if lex.err != nil {
					return lex.err
				}
				if !strings.HasSuffix(lex.token, "]") {
					return lex.Errorf("unclosed brackets, closing ']' not found in body declaration")
				}

				for i, v := range lex.token {
					// a non-space character found inbetween brackets
					if v != '[' && v != ']' && !unicode.IsSpace(v) {
						fmt.Printf("WARNING: Syncing data within datatable syntax is not currently supported. The following contents will be ignored:\n%s\n", lex.token[i:len(lex.token)-1])
						break
					}
				}

				return nil
			default:
				panic(fmt.Sprintf("unsupported decl type: %d", decl.declType))
			}
		default:
			switch decl.declType {
			case functionType:
				for lex.readLine() {
				}
				if lex.err != nil {
					return lex.err
				}
			}

			return nil
		}
		decl.parseState += 1
	}
}

func parseSignature(l *lexer, decl *declaration) error {
	if !strings.HasPrefix(l.tokenBuf.String(), "(") {
		switch decl.declType {
		case functionType:
			return l.Errorf("expected '(' after '=' in let statement, i.e. let myFunc = (arg1:string){}")
		case tableType:
			return l.Errorf("expected '(' after 'datatable' in let statement, i.e. let myTable = datatable (['foo']:string) [], found: %s", l.tokenBuf.String())
		}
	}

	l.consumeTill(')')
	if l.err != nil {
		return l.err
	}

	if !strings.HasSuffix(l.token, ")") {
		return l.Errorf("unmatched parenthesis in method signature, missing closing ')'")
	}

	decl.signature = l.token
	return nil
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
				".create-or-alter function with (folder=\"%s\",docstring=\"%s\") %s %s\n",
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
