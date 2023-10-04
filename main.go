package bibsin

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/scanner"
)

type NodeType uint8

// Node types.
const (
	Root NodeType = iota
	Record
	Field
)

func (t NodeType) String() string {
	switch t {
	case Root:
		return "Root"
	case Record:
		return "Record"
	case Field:
		return "Field"
	default:
		return fmt.Sprintf("NodeType(%d)", t)
	}
}

type Node struct {
	Parent   *Node
	Children []*Node
	Type     NodeType
	Key      string
	Value    string
}

func (n *Node) addChild(c *Node) {
	n.Children = append(n.Children, c)
}

func Print(w io.Writer, n *Node) error {
	//FIXME: check for errors
	switch n.Type {
	case Root:
		for _, c := range n.Children {
			Print(w, c)
		}
	case Record:
		fmt.Fprintf(w, "\n@%s{%s,\n", n.Key, n.Value)
		for i, c := range n.Children {
			Print(w, c)
			if i < len(n.Children) {
				fmt.Fprintln(w, ",")
			}
		}
		fmt.Fprintln(w, "}")
	case Field:
		fmt.Fprintf(w, "%s={%s}", n.Key, n.Value)
	default:
		return fmt.Errorf("Unknown NodeType(%d)", n.Type)
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

// imports Google scholar bibtex export
func Parse(r io.Reader, fileName string) (*Node, error) {
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

	root := &Node{Type: Root}
	node := root
	var scanErr error
	var s scanner.Scanner
	result := func(msg string) (*Node, error) {
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
		fmt.Printf("%s: %s\n", s.Position, s.TokenText())
		if tok == EOF {
			//TODO: check for unexpected eof
			break
		}
		if scanErr != nil {
			return root, scanErr
		}
		// parsing depends on location in file
		switch node.Type {
		case Root:
			// only allowing records
			if tok != AT {
				return result("expected a record")
			}
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
			node = &Node{Type: Record, Key: citeType, Value: citeKey}
		case Record:
			// field started
			if tok != IDENTIFIER {
				return result("expected field name")
			}
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
			switch s.Scan() {
			case COMMA:
				node.addChild(&Node{Type: Field, Key: fieldName, Value: b.String()})
			case RBRACE: //end of record
				node.addChild(&Node{Type: Field, Key: fieldName, Value: b.String()})
				root.addChild(node)
				node = root
			default:
				return result("expected , or }")
			}
		}
	}
	return root, nil
}
