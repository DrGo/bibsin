package bibsin

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

type Node interface {
	IsRoot() bool
	Line() int
	Key() string
	Value() string
	BibtexRepr() string
	Children() []Node
}

type Record struct {
	children []Node
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

func (n *Record) Children() []Node {
	return n.children
}

func (n *Record) addChild(c Node) {
	n.children = append(n.children, c)
}

func (n *Record) Field(fieldName string) string {
	for _, fld := range n.children {
		fld := fld.(*Field)
		if fld.key == fieldName {
			return fld.value
		}
	}
	return ""
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
	return fmt.Sprintf("%s={%s}", rec.key, rec.value)
}

func (n *Field) Children() []Node {
	return nil
}

func newRoot(fileName string) *Record {
	return &Record{key: "__ROOT__", value: fileName}
}

func Print(w io.Writer, n Node) error {
	//FIXME: check for errors
	switch n := n.(type) {
	case *Record:
		if n.IsRoot() {
			for _, c := range n.children {
				Print(w, c)
			}
			return nil
		}
		// record
		fmt.Fprintf(w, n.BibtexRepr())
		for i, c := range n.children {
			Print(w, c)
			if i < len(n.children) {
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

// BNF
// Database        ::= (Junk '@' Entry)*
// Junk                ::= .*?
// Entry        ::= Record
//                |   Comment
//                |   String
//                |   Preamble
// Comment        ::= "comment" [^\n]* \n                -- ignored
// String        ::= "string" '{' Field* '}'
// Preamble        ::= "preamble" '{' .* '}'         -- (balanced)
// Record        ::= Type '{' Key ',' Field* '}'
//                |   Type '(' Key ',' Field* ')' -- not handled
// Type                ::= Name
// Key                ::= Name
// Field        ::= Name '=' Value
// Name                ::= [^\s\"#%'(){}]*
// Value        ::= [0-9]+
//                |   '"' ([^'"']|\\'"')* '"'
//                |   '{' .* '}'                         -- (balanced)

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
	root := newRoot(fileName)
	currentNode := root
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
	tok := s.Scan() //get the first token 
loop:
	for {
		// parsing depends on location in file; records only legal at the highest level 
		if currentNode.IsRoot() {
			//ignore anything that does not start by @
			for ; tok != EOF && tok != AT; tok = s.Scan() {
			}
			// fmt.Printf("%s: %s\n", s.Position, s.TokenText())
			if tok == EOF {
				//TODO: check for unexpected eof
				break loop
			}
			if scanErr != nil {
				return root, scanErr
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
			//create a new record and set as current node
			currentNode = &Record{key: citeKey, value: citeType, line: s.Pos().Line}
			tok = s.Scan()
			continue
		}
		// we must be in a record so parse fields 
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
		// start parsing value 
		b.Reset()
		level := 0
	valueLoop:
		for tok = s.Next(); tok != EOF; tok = s.Next() {
			switch tok {
			case EOF:				
				return result("unterminated value")	
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
		// comma is optional before the record's closing RBRACE,
		// but here always optional
		// fmt.Println(s.TokenText(), s.Peek())
		fld := &Field{key: fieldName, value: b.String(), line: lineNum}
		switch s.Scan() {
		case COMMA:
			currentNode.addChild(fld)
		case RBRACE: //parsed the last field, switch current node to root  
			currentNode.addChild(fld)
			root.addChild(currentNode)
			currentNode = root
		default:
			return result("expected , or }")
		}
		tok = s.Scan()
	} //for
	return root, nil
}

type SetActionType int8

const (
	SetNoAction SetActionType = iota
	// SetIntersection finds records common to one or more sets and
	// returns the record that belongs to the first set
	// if one file, SetIntersection results in a set that includes the first record
	SetIntersection
	SetUnion
)

type NodeInfo struct {
	Node   Node
	Parent Node
}

type DedupMap = map[string][]NodeInfo

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
	for idxTerm, nodes := range err.DuplicateSet { //
		if ndup := len(nodes); ndup > 1 {
			sb.WriteString(fmt.Sprintf("[%s] has %d occurrences in lines \n", idxTerm, ndup))
			for _, n := range nodes {
				// write filename: line
				sb.WriteString(fmt.Sprintf("%s:%d\n", n.Parent.Value(), n.Node.Line()))
			}
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// func Deduplicate(root Node, action DedupActionType) error {
// 	if !root.IsRoot() {
// 		return fmt.Errorf("cannot deduplicate non-root nodes")
// 	}
// 	rn := root.(*Record)
// 	n := len(rn.children)
// 	if n == 0 {
// 		return fmt.Errorf("empty list")
// 	}
// 	set := make(DedupMap, n)
// 	for _, c := range rn.children {
// 		set[c.Key()] = append(set[c.Key()], c)
// 	}
// 	duplicateSets := 0
// 	for _, nodes := range set {
// 		if len(nodes) > 1 {
// 			duplicateSets++
// 		}
// 	}
// 	if action == DedupReport {
// 		if duplicateSets == 0 {
// 			return nil
// 		}
// 		return DedupError{DuplicateSetCount: duplicateSets, DuplicateSet: set}
// 	}

// 	return nil
// }

func DeduplicateByContents(nodes []Node, fldNames []string, action SetActionType) (Node, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("nothing to deduplicate")
	}
	var sb strings.Builder
	indexEntry := func(n Node, fldNames []string) string {
		rec := n.(*Record)
		sb.Reset()
		for _, fldname := range fldNames {
			sb.WriteString(onlyASCIAlphaNumeric(rec.Field(fldname)))
		}
		return sb.String()
	}

	set := make(DedupMap, len(nodes[0].Children())*len(nodes))
	for _, r := range nodes {
		for _, c := range r.Children() {
			idx := indexEntry(c, fldNames)
			set[idx] = append(set[idx], NodeInfo{c, r})
		}
	}

	duplicateSets := 0
	for _, nodes := range set {
		if len(nodes) > 1 {
			duplicateSets++
		}
	}
	if action == SetNoAction {
		if duplicateSets == 0 {
			return nil, nil
		}
		return nil, DedupError{DuplicateSetCount: duplicateSets, DuplicateSet: set}
	}
	if action == SetIntersection {
		if duplicateSets == 0 {
			return nil, fmt.Errorf("no common records")
		}
		res := newRoot("intersection.bib")
		for _, nodes := range set {
			if ndup := len(nodes); ndup > 1 { //duplicates
				res.addChild(nodes[0].Node)
			}
		}
		return res, nil
	}
	if action == SetUnion {
		res := newRoot("union.bib")
		for _, nodes := range set {
			res.addChild(nodes[0].Node)
		}
		return res, nil
	}
	return nil, fmt.Errorf("invalid set action")
}

func lower(ch rune) rune { return ('a' - 'A') | ch } // returns lower-case ch iff ch is ASCII letter
func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}
func ByteSlice2String(bs []byte) string {
	if len(bs) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(bs), len(bs))
}

func isASCIIAlphaNumeric(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || '0' <= ch && ch <= '9'
}
func onlyASCIAlphaNumeric(s string) string {
	b := make([]byte, len(s))
	i := 0
	for _, ch := range s {
		ch := lower(ch)
		if isASCIIAlphaNumeric(ch) {
			b[i] = byte(ch)
			i++
		}
	}
	return ByteSlice2String(b[:i])
}

//translations for latex escapes
//         _latex.Add("\\`{o}", "ò");
//         _latex.Add("\\'{o}", "ó");
//         _latex.Add("\\^{o}", "ô");
//         _latex.Add("\\\"{o}", "ö");
//         _latex.Add("\\H{o}", "ő");
//         _latex.Add("\\~{o}", "õ");
//         _latex.Add("\\c{c}", "ç");
//         _latex.Add("\\k{a}", "ą");
//         _latex.Add("\\l{}", "ł");
//         _latex.Add("\\={o}", "ō");
//         _latex.Add("\\b{o}", "o");
//         _latex.Add("\\.{o}", "ȯ");
//         _latex.Add("\\d{u}", "ụ");
//         _latex.Add("\\r{a}", "å");
//         _latex.Add("\\u{o}", "ŏ");
//         _latex.Add("\\v{s}", "š");
//         _latex.Add("\\t{oo}", "o͡o");
//         _latex.Add("\\o", "ø");

//         _latex.Add("\\%", "%");
//         _latex.Add("\\$", "$");
//         _latex.Add("\\{", "{");
//         _latex.Add("\\_", "_");
//         _latex.Add("\\P", "¶");
//         _latex.Add("\\ddag", "‡");
//         _latex.Add("\\textbar", "|");
//         _latex.Add("\\textgreater", ">");
//         _latex.Add("\\textendash", "–");
//         _latex.Add("\\texttrademark", "™");
//         _latex.Add("\\textexclamdown", "¡");
//         _latex.Add("\\textsuperscript{ a}", "a");
//         _latex.Add("\\pounds", "£");
//         _latex.Add("\\#", "#");
//         _latex.Add("\\&", "&");
//         _latex.Add("\\}", "}");
//         _latex.Add("\\S", "§");
//         _latex.Add("\\dag", "†");
//         _latex.Add("\\textbackslash", "\\");
//         _latex.Add("\\textless", "<");
//         _latex.Add("\\textemdash", "—");
//         _latex.Add("\\textregistered", "®");
//         _latex.Add("\\textquestiondown", "¿");
//         _latex.Add("\\textcircled{ a}", "ⓐ");
//         _latex.Add("\\copyright", "©");

//         _latex.Add("$\\backslash$", "\\");
//         _latex.Add("\\'{e}", "ë");
//         _latex.Add("\\'{i}", "ë");
