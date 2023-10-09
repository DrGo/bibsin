package bibsin

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	LPAREN     rune = '('
	RPAREN     rune = ')'
	LBRACE     byte = '{'
	RBRACE     byte = '}'
	LBRACK     rune = '['
	RBRACK     rune = ']'
	COMMA      byte = ','
	COLON      rune = ':'
	SEMICOLON  rune = ';'
	EQUAL      byte = '='
	AT         byte = '@'
)

type Options struct {
}

// Parse parses a Google scholar bibtex export provided as io.Reader or
// a name of a file.
func Parse(r io.Reader, fileName string, opts Options) (Node, error) {
	if r == nil {
		if fileName == "" {
			return nil, fmt.Errorf("nothign to parse")
		}
		f, err := os.Open(fileName)
		if err != nil {
			return nil, fmt.Errorf("can't process file %s: %w", fileName, err)
		}
		defer f.Close()
		r = f
	}
	return newParser(r, fileName, opts).parse()
}

type parser struct {
	r         *bufio.Reader
	fileName  string
	rawBuffer []byte // using for reading
	lineNum   int
	// offset is the input stream byte offset of the current reader position.
	offset int64
}

func newParser(r io.Reader, fileName string, opts Options) *parser {
	return &parser{
		r:        bufio.NewReaderSize(r, 2048),
		fileName: fileName,
	}
}

// readLine reads the next line without the trainling EOF marker(S)
// If EOF is hit without a trailing endline, it will be omitted.
// If some bytes were read, then the error is never io.EOF.
// The result is only valid until the next call to readLine.
func (p *parser) readLine() ([]byte, error) {
	line, err := p.r.ReadSlice('\n')
	if err == bufio.ErrBufferFull {
		p.rawBuffer = append(p.rawBuffer[:0], line...)
		for err == bufio.ErrBufferFull {
			line, err = p.r.ReadSlice('\n')
			p.rawBuffer = append(p.rawBuffer, line...)
		}
		line = p.rawBuffer
	}
	readSize := len(line)
	if readSize > 0 && err == io.EOF {
		err = nil
		if line[readSize-1] == '\r' {
			line = line[:readSize-1]
		}
	}
	p.lineNum++
	p.offset += int64(readSize)
	// Normalize \r\n to \n on all input lines.
	if n := len(line); n >= 2 && line[n-2] == '\r' && line[n-1] == '\n' {
		line[n-2] = '\n'
		line = line[:n-1]
	}
	return line, err
}

// getRune returns the next rune in b or utf8.RuneError.
// func getRune(b []byte) rune {
// 	r, _ := utf8.DecodeRune(b)
// 	return r
// }

func (p *parser) parse() (Node, error) {
	root := newRoot(p.fileName)
	var (
		scanErr error
		line    []byte
	)
	result := func(msg string) (Node, error) {
		//TODO: add error msgs
		return root, fmt.Errorf("parsing error at %d: %s", p.lineNum, msg)
	}
	currentNode := root
	ignored := false 
mainloop:
	for scanErr == nil {
		line, scanErr = p.readLine()
		// fmt.Printf("%t %t %s \n", ignored, currentNode.IsRoot(), string(line))	
		if len(line) == 0 { //error or empty line or junk line
			line = nil
			continue mainloop
		}
		// we have text
		//TODO: process commands 'comment' and 'premable' read value
		// 'string' (a macro) read value and store in a map
		//TODO: allow use of ( instead of {; set righ delimiter to ) or }
		switch {
		case line[0] == AT && !currentNode.IsRoot():
			return result("invalid @; possibly record missing line starting with }")
		case line[0] == AT:
			// we are at root and we have a line starting with @, parse a record header
			// -> recordtype { citekey, which can be anything till comma or whitespace
			idx := bytes.IndexByte(line, LBRACE)
			if idx == -1 {
				return result("{ is missing")
			}
			//ignore non-article entries 
			typ:= strings.ToLower(trimAffixes(line[1:idx],true))
			if typ == "comment" || typ == "preamble" || typ == "string" {
				if bytes.IndexByte(line, RBRACE) == -1 {   // not a one liner 
				ignored = true 
				}				
				continue mainloop
			} 
			ignored = false 
			//create a new record and set as current node
			currentNode = &Record{
				key:trimAffixes(line[idx+1:], false), 
				value: typ,                                //record type
				line:  p.lineNum}
			// continue mainloop
		case line[0] == RBRACE && ignored:
			ignored = false 
		case line[0] == RBRACE && currentNode.IsRoot():
			return result("} outside a record")
		case line[0] == RBRACE:
			//add node to root and switch to root
			root.addChild(currentNode)
			currentNode = root
		default:
			if ignored || currentNode.IsRoot() { // text directly under root or ignored
				// line = nil
				continue mainloop
			}
			// we are in a record so parse } or fields: fldname= value,
			idx := bytes.IndexByte(line, EQUAL)
			if idx == -1 {
				return result("= is missing")
			}
			fldname:= trimAffixes(line[:idx], true)
			value:= trimAffixes(line[idx+1:], false)
			fld := &Field{
				value: string(value),
				key:  fldname, 
				line:  p.lineNum}
			// fmt.Printf("%s\n", fld.value)
			currentNode.addChild(fld)
			// comma is normally optional before the record's closing RBRACE,
			// but here always it is always optional
		}
	} //for
	return root, nil
}

