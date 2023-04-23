package ksd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
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

type lexer struct {
	r        *bufio.Reader
	token    string
	tokenBuf strings.Builder

	err error
	row int
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
	if line == "" && err == io.EOF {
		return false
	}

	if err != io.EOF && err != nil {
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
// the delim byte is saved into tokenBuf.
func (l *lexer) readSpaced(s rune) (hasMore bool) {
	hasMore = l.readSpacedFunc(func(r rune) bool {
		return r == s
	})
	return
}

// delimFunc is used in functions like readSpaceFunc
// that returns true if the rune present is a
// delimiter.
type delimFunc func(s rune) bool

// advances the cursor to one-past the first occurence of the delimiter
// as specified by delimFunc.
//
// whitespace is ignored and discarded when advancing the cursor.
// if non-whitespace, non-delim char is found, errNonDelimWhitespaceFound is returned.
//
// the delimiter is saved into tokenBuf.
func (l *lexer) readSpacedFunc(f delimFunc) (hasMore bool) {
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
		} else if f(r) {
			_, err = l.tokenBuf.WriteRune(r)
			if err != nil {
				l.err = err
				return false
			}
			return true
		}

		l.err = errNonDelimWhitespaceFound
		return false
	}
}

type ParseError struct {
	row int
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

func parse(reader io.ReadSeeker) (*declaration, error) {
	lexer := newLexer(bufio.NewReader(reader))
	letFound := false
	comments := make([]string, 0, 20)
	offset := int64(0)
	for lexer.consumeLine() {
		line := lexer.token
		if strings.HasPrefix(strings.TrimLeftFunc(line, unicode.IsSpace), "let") {
			letFound = true
			break
		}

		comments = append(comments, line)
		offset += int64(len(line))
	}

	if !letFound {
		return nil, lexer.Errorf("parsing file: missing 'let' statement in file")
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

	_, err := reader.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("seeking file: %w", err)
	}
	lexer.row -= 1
	lexer.col = 0
	lexer.r.Reset(reader)
	lexer.token = ""
	lexer.tokenBuf.Reset()

	decl := &declaration{doc: docstring}
	err = parseDeclaration(lexer, decl)
	if err != nil {
		return nil, fmt.Errorf("parsing declaration: %w", err)
	}
	return decl, nil
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
		case 1: // \s+name\s*'='
			lex.skipSpace()
			lex.consumeTill('=')
			if lex.err != nil {
				return lex.err
			}

			if !strings.HasSuffix(lex.token, "=") {
				return lex.Errorf("expected variable assignment '=' after identifier")
			}

			// letters, digits, underscores (_), spaces, dots (.), and dashes (-).
			// Identifiers consisting only of letters, digits, and underscores don't require quoting when the identifier is being referenced.
			//Identifiers containing at least one of (spaces, dots, or dashes) do require quoting (see below).
			isIdentifier := func(r rune) bool {
				return unicode.IsLetter(r) ||
					unicode.IsDigit(r) ||
					r == '_'
			}
			var name strings.Builder
			for _, v := range lex.token[:len(lex.token)-1] {
				if isIdentifier(v) {
					name.WriteRune(v)
					continue
				}

				if !unicode.IsSpace(v) {
					return lex.Errorf("unexpected character '%s'", string(v))
				}
			}

			decl.name = name.String()
		case 2: // \s*[datatable]\s*({signature})
			found := lex.readSpacedFunc(func(s rune) bool {
				return string(s) == "d" || string(s) == "("
			})
			if lex.err == errNonDelimWhitespaceFound {
				return lex.Errorf("expected '(' for function declaration, or 'datatable' for table declaration")
			} else if lex.err != nil {
				return lex.err
			} else if !found {
				return lex.Errorf("expected '(' for function declaration, or 'datatable' for table declaration")
			}

			switch lex.tokenBuf.String() {
			case "d":
				decl.declType = tableType
				lex.consumeTill('(')
				if lex.err != nil {
					return lex.err
				}

				if !strings.HasPrefix(lex.token, "datatable") {
					return lex.Errorf("invalid keyword. expected 'datatable' for table declaration")
				}

				start := len("datatable")
				for i, v := range lex.token[start : len(lex.token)-1] {
					if !unicode.IsSpace(v) {
						lex.col = lex.col + (i - len(lex.token))
						return lex.Errorf("invalid character '%s'", string(v))
					}
				}

				_, err := lex.tokenBuf.WriteString("(")
				if err != nil {
					return err
				}
			case "(":
				decl.declType = functionType
				// we currently do not parse the signature due to language complications,
				// instead, everything is allowed through
				// and stored as the body of the function.
				for lex.readLine() {
				}
				if lex.err != nil {
					return lex.err
				}

				lex.consumeToken()
				decl.body = lex.token
				return nil
			}

			found = lex.consumeTill(')')
			if lex.err != nil {
				return lex.err
			} else if !found {
				return lex.Errorf("unmatched parenthesis, missing ')' in declaration signature")
			}

			decl.signature = lex.token
		// Table: \s*[\s*]
		case 3:
			switch decl.declType {
			case tableType:
				found := lex.readSpaced('[')
				if lex.err == errNonDelimWhitespaceFound {
					return lex.Errorf("unexpected character found. expected '[' for beginning of table body")
				} else if lex.err != nil {
					return lex.err
				} else if !found {
					return lex.Errorf("expected '[' for beginning of table body")
				}

				found = lex.consumeTill(']')
				if lex.err != nil {
					return lex.err
				} else if !found {
					return lex.Errorf("unmatched brackets, missing ']' for end of table body")
				}

				for i, v := range lex.token {
					// a non-space character found inbetween brackets
					if v != '[' && v != ']' && !unicode.IsSpace(v) {
						fmt.Printf("WARNING: Syncing data within datatable syntax is not currently supported. The following contents will be ignored:\n%s\n", lex.token[i:len(lex.token)-1])
						break
					}
				}
				decl.body = ""
				return nil
			default:
				panic(fmt.Sprintf("unsupported declaration type: %d", decl.declType))
			}
		default:
			return nil
		}
		decl.parseState += 1
	}
}

func write(
	writer io.Writer,
	decl *declaration,
	folder string) error {
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
