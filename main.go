package bibsin

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/scanner"
)

type Node interface {
	IsRoot() bool
	Line() int
	Key() string
	Value() string
	BibtexRepr() string 
}

// Node types.
// const (
// 	Root NodeType = iota
// 	Record
// 	Field
// )

// func (t NodeType) String() string {
// 	switch t {
// 	case Root:
// 		return "Root"
// 	case Record:
// 		return "Record"
// 	case Field:
// 		return "Field"
// 	default:
// 		return fmt.Sprintf("NodeType(%d)", t)
// 	}
// }

type Record struct {
	Children []Node
	key      string // citation key; ROOT for root node
	value    string // bibtex type
	line     int
}

func (rec *Record) IsRoot() bool {
	return rec.key == "__ROOT__"
}

func (rec *Record) Line() int {
	return rec.line
}

func (rec *Record) Key() string {
	return rec.key
}
func (rec *Record) Value() string {
	return rec.value
}

func (rec *Record) BibtexRepr() string {
	return fmt.Sprintf("\n@%s{%s,\n", rec.key, rec.value)
}

func (n *Record) addChild(c Node) {
	n.Children = append(n.Children, c)
}

type Field struct {
	key   string // name of field
	value string // value of field
	line  int
}

func (rec *Field) IsRoot() bool {
	return false
}

func (rec *Field) Line() int {
	return rec.line
}

func (rec *Field) Key() string {
	return rec.key
}
func (rec *Field) Value() string {
	return rec.value
}

func (rec *Field) BibtexRepr() string {
	return fmt.Sprintf( "%s={%s}", rec.key, rec.value)
}

func Print(w io.Writer, n Node) error {
	//FIXME: check for errors
	switch n := n.(type) {
	case *Record:
		if n.IsRoot() {
			for _, c := range n.Children {
				Print(w, c)
			}
			return nil
		}
		// record
		fmt.Fprintf(w, n.BibtexRepr())
		for i, c := range n.Children {
			Print(w, c)
			if i < len(n.Children) {
				fmt.Fprintln(w, ",")
			}
		}
		fmt.Fprintln(w, "}")
	case *Field:
		fmt.Fprintf(w, n.BibtexRepr())
	default:
		return fmt.Errorf("Unknown Node type")
	}
	return nil
}

type Entry map[string]string

const (
	EOF        rune = scanner.EOF
	IDENTIFIER rune = scanner.Ident
	LPAREN     rune = '('
	RPAREN     rune = ')'
	LBRACE     rune = '{'
	RBRACE     rune = '}'
	LBRACK     rune = '['
	RBRACK     rune = ']'
	COMMA      rune = ','
	COLON      rune = ':'
	SEMICOLON  rune = ';'
	EQUAL      rune = '='
	AT         rune = '@'
)

// comment = [^@]*
// ws = [ \t\n]*
// ident = ![0-9] (![ \t"#%'(),={}] [\x20-\x7f])+
// command_or_entry = '@' ws (comment / preamble / string / entry)
// comment = 'comment'
// preamble = 'preamble' ws ( '{' ws preamble_body ws '}'
//                          / '(' ws preamble_body ws ')' )
// preamble_body = value
// string = 'string' ws ( '{' ws string_body ws '}'
//                      / '(' ws string_body ws ')' )
// string_body = ident ws '=' ws value
// entry = ident ws ( '{' ws key ws entry_body? ws '}'
//                  / '(' ws key_paren ws entry_body? ws ')' )
// key = [^, \t}\n]*
// key_paren = [^, \t\n]*
// entry_body = (',' ws ident ws '=' ws value ws)* ','?
// value = piece (ws '#' ws piece)*
// piece
//     = [0-9]+
//     / '{' balanced* '}'
//     / '"' (!'"' balanced)* '"'
//     / ident
// balanced
//     = '{' balanced* '}'
//     / [^{}]

type Options struct {
}

// Parse parses a Google scholar bibtex export provided as io.Reader or
// a name of a file.
func Parse(r io.Reader, fileName string, opt Options) (Node, error) {
	if r == nil {
		var err error
		if fileName == "" {
			return nil, fmt.Errorf("nothign to parse")
		}
		r, err = os.Open(fileName)
		if err != nil {
			return nil, fmt.Errorf("can't process file %s: %w", fileName, err)
		}
	}
	root := &Record{key: "__ROOT__"}
	node := root
	var scanErr error
	var s scanner.Scanner
	result := func(msg string) (Node, error) {
		//TODO: add error msgs
		return root, fmt.Errorf("parsing error at %s %s but found %s", s.Pos(), msg, s.TokenText())
	}
	s.Init(r)
	s.Mode = scanner.ScanIdents | scanner.ScanStrings
	s.Error = func(s *scanner.Scanner, msg string) {
		scanErr = fmt.Errorf("parsing error at %s: %s", s.Pos(), msg)
	}
	// s.Whitespace ^=  1<<'\n' //do not skip new lines
	s.Filename = fileName
	var b strings.Builder
loop:
	for {
		tok := s.Scan()
		// fmt.Printf("%s: %s\n", s.Position, s.TokenText())
		if tok == EOF {
			//TODO: check for unexpected eof
			break
		}
		if scanErr != nil {
			return root, scanErr
		}
		// parsing depends on location in file
		if node.IsRoot() {
			// only allowing records
			if tok != AT {
				return result("expected a record")
			}
			lineNum := s.Pos().Line
			// scan the commdand/entry type identifier
			if s.Scan() != IDENTIFIER {
				return result("expected identifier")
			}
			citeType := s.TokenText()
			//TODO: reject commands
			// only accepting database entries
			if s.Scan() != LBRACE {
				return result("expected {")
			}
			// scan the commdand/entry type identifier
			if s.Scan() != IDENTIFIER {
				return result("expected citation key")
			}
			citeKey := s.TokenText()
			if s.Scan() != COMMA {
				return result("expected ,")
			}
			//create a new  node
			node = &Record{key: citeKey, value: citeType, line: lineNum}
			continue
		}
		// field started
		if tok != IDENTIFIER {
			return result("expected field name")
		}
		lineNum := s.Pos().Line
		fieldName := s.TokenText()
		if s.Scan() != EQUAL {
			return result("expected =")
		}
		if s.Scan() != LBRACE {
			return result("expected {")
		}
		b.Reset()
		level := 0
	valueLoop:
		for tok = s.Next(); tok != EOF; tok = s.Next() {
			switch tok {
			case LBRACE:
				level++
			case RBRACE:
				if level == 0 {
					break valueLoop
				}
				level--
				if level < 0 {
					return nil, fmt.Errorf("} without matching { around %s", s.Pos())
				}
			default:
				b.WriteRune(tok)
			}
		}
		//either next token is EOF or RBRACE
		if tok == EOF {
			continue loop
		}
		// comma is optional before the record's closing RBRACE,
		// but here always optional
		// fmt.Println(s.TokenText(), s.Peek())
		fld := &Field{key: fieldName, value: b.String(), line: lineNum}
		switch s.Scan() {
		case COMMA:
			node.addChild(fld)
		case RBRACE: //end of record
			node.addChild(fld)
			root.addChild(node)
			node = root
		default:
			return result("expected , or }")
		}
	} //for
	return root, nil
}

type DedupActionType int8

const (
	DedupReport DedupActionType = iota
	DedupKeepFirst
	DedupUpdateKeys
)

type DedupMap = map[string][]Node
type DedupError struct {
	DuplicateSetCount int
	DuplicateSet      DedupMap
}

func (err DedupError) Error() string {
	if err.DuplicateSetCount == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(strconv.Itoa(err.DuplicateSetCount) + " duplicate sets found\n")
	for _, nodes := range err.DuplicateSet {
		if ndup := len(nodes); ndup > 1 {
			sb.WriteString(fmt.Sprintf("[%s] has %d occurrences in lines \n", nodes[0].Key(), ndup))
			for _, n := range nodes {
				sb.WriteString(strconv.Itoa(n.Line()) + " ")
			}
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func Deduplicate(root Node, action DedupActionType) error {
	if !root.IsRoot() {
		return fmt.Errorf("cannot deduplicate non-root nodes")
	}
	rn := root.(*Record)
	n := len(rn.Children)
	if n == 0 {
		return fmt.Errorf("empty list")
	}
	set := make(DedupMap, n)
	for _, c := range rn.Children {
		set[c.Key()] = append(set[c.Key()], c)
	}
	if action == DedupReport {
		duplicateSets := 0
		for _, nodes := range set {
			if len(nodes) > 1 {
				duplicateSets++
			}
		}
		if duplicateSets == 0 {
			return nil
		}
		return DedupError{DuplicateSetCount: duplicateSets, DuplicateSet: set}
	}
	return nil
}
