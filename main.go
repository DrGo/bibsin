package bibsin

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"text/scanner"
	"unicode"
)

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

type Options struct {
}

// Parse parses a Google scholar bibtex export provided as io.Reader or
// a name of a file.
func Parse(r io.Reader, fileName string, opt Options) (Node, error) {
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
			// we must have parsed @
			if s.Pos().Column> 1 {
				continue loop // only relevant if it is the first char
			}
			// scan the commdand/entry type identifier
			if s.Scan() != IDENTIFIER {
				return result("expected identifier")
			}
			citeType := s.TokenText()

			if s.Scan() != LBRACE {
				return result("expected {")
			}
			// read citekey which can be anything till comma or whitespace
			b.Reset()
		refLoop:
			for tok = s.Next(); ; tok = s.Next() {
				switch {
				case tok == EOF:
					return result("unterminated key")
				case tok == COMMA:
					break refLoop
				case unicode.IsSpace(tok):
					break refLoop
				default:
					b.WriteRune(tok)
				}
			} //refLoop
			citeKey := b.String()
			//TODO: process commands 'comment' and 'premable' read value
			// 'string' (a macro) read value and store in a map
			//TODO: allow use of ( instead of {; set righ delimiter to ) or }
			// only accepting database entries
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
		//TODO: allow " instead of { and if the entire value is digits no delimiter needed
		//algo: check for { " or digit
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
		} //valueLoop
		// fmt.Println(s.TokenText(), s.Peek())
		fld := &Field{key: fieldName, value: b.String(), line: lineNum}
		// comma is normally optional before the record's closing RBRACE,
		// but here always it is always optional
		tok = s.Scan() // expecting , or } or ,}
		if tok == COMMA {
			currentNode.addChild(fld)
			tok = s.Scan()
			if tok == RBRACE { //parsed the last field, switch current node to root
				root.addChild(currentNode)
				currentNode = root
			} else {
				continue loop // new field
			}
		} else if tok == RBRACE { // field with no comma
			currentNode.addChild(fld)
			root.addChild(currentNode)
			currentNode = root
		} else {
			return result("expected , or }")
		}
	} //for
	return root, nil
}

type SetActionType int8

const (
	SetNoAction SetActionType = iota
	// SetIntersect finds records common to one or more sets and
	// returns the record that belongs to the first set
	// if one file, SetIntersect results in a set that includes the first record
	SetIntersect
	SetUnion
	SetConcat
)

type NodeInfo struct {
	Node   *Record
	Parent *Record
}

type DedupMap = map[string][]NodeInfo

type DedupReport struct {
	DuplicateSetCount int
	DuplicateSet      DedupMap
	ResultSetCount    int
}

func (dr DedupReport) Print(w io.Writer) (err error) {
	if dr.DuplicateSetCount == 0 {
		return nil
	}
	fmt.Fprintf(w, "%d duplicate sets found\n", dr.DuplicateSetCount)
	for idxTerm, nodes := range dr.DuplicateSet { //
		if ndup := len(nodes); ndup > 1 {
			_, err = fmt.Fprintf(w, "%s\n[%s] has %d occurrences in lines \n", strings.Repeat("*", 60), idxTerm, ndup)
			for _, n := range nodes {
				// write filename: line
				_, err = fmt.Fprintf(w, "%s:%d\n", n.Parent.Value(), n.Node.Line())
				err = Print(w, n.Node)
			}
		}
	}
	if err != nil {
		fmt.Printf("%d records processed\n", dr.ResultSetCount)
	}
	return err
}

func (dr DedupReport) String() string {
	var b = new(bytes.Buffer)
	if err:= dr.Print(b); err != nil {
		b.WriteString("error: "+ err.Error())
	}
	return b.String()
}

// func Query(nodes []Node, query string)(Node, error) {
// 	for _, r := range nodes {
// 		for _, c := range r.Children() {
// }
// indexEntry returns a string concating values of fields
func indexEntry(n Node, fldNames []string, raw bool) string {
	var sb strings.Builder
	rec := n.(*Record)
	for _, fldname := range fldNames {
		sb.WriteString(rec.Field(fldname))
	}
	if raw {
		return sb.String()
	}
	return onlyASCIAlphaNumeric(sb.String())
}

// Deduplicate performs various set operations on one or more ref sets
// using the concatinated values of field names. If no fields specified,
// citekey is used to deduplicate the set.
// if no error encountered, it returns a DedupReport struct if action== SetNoAction
// and additionally a set of processed refs if action != SetNoAction
func Deduplicate(nodes []Node, fldNames []string, action SetActionType) (Node, *DedupReport, error) {
	if len(nodes)*len(nodes[0].Children()) == 0 {
		return nil, nil, fmt.Errorf("nothing to deduplicate")
	}
	hasFields := len(fldNames) == 0
	citekey := !hasFields || slices.Contains(fldNames, "citekey")
	dupSet := make(DedupMap, len(nodes[0].Children())*len(nodes))
	for _, r := range nodes {
		for _, c := range r.Children() {
			idx := ""
			if hasFields {
				indexEntry(c, fldNames, false)
			}
			if citekey {
				idx = idx + c.Key()
			}
			dupSet[idx] = append(dupSet[idx], NodeInfo{c.(*Record), r.(*Record)})
		}
	}
	duplicateSets := 0
	for _, nodes := range dupSet {
		if len(nodes) > 1 {
			duplicateSets++
		}
	}
	dr := &DedupReport{DuplicateSetCount: duplicateSets, DuplicateSet: dupSet}
	if action == SetNoAction {
		return nil, dr, nil
	}
	if action == SetIntersect {
		if duplicateSets == 0 {
			return nil, nil, fmt.Errorf("no common records")
		}
		res := newRoot("intersection.bib")
		for _, nodes := range dupSet {
			if ndup := len(nodes); ndup > 1 { //duplicates
				res.addChild(nodes[0].Node) //print the first in the set
				dr.ResultSetCount++
			}
		}
		return res, dr, nil
	}
	if action == SetUnion {
		res := newRoot("union.bib")
		for _, nodes := range dupSet {
			res.addChild(nodes[0].Node)
			dr.ResultSetCount++
		}
		return res, dr, nil
	}
	return nil, nil, fmt.Errorf("invalid set action")
}

// ValidKeys checks if all records have citekeys and all are unique
func ValidKeys(n Node) bool {
	_, dr, err := Deduplicate([]Node{n}, []string{}, SetNoAction)
	if err != nil {
		return true // only error is nothign to duplicate
	}
	return dr.DuplicateSetCount == 0
}

// ScholarCiteKey generates a new key using last name of the first author + pub year+
// first word of the title + first letter of article type + page or volume #
func NewCiteKey(rec *Record) string{
	var sb strings.Builder
	word, _,_ :=strings.Cut(rec.Field("author"), ",")
	sb.WriteString(strings.ToLower(word))
	sb.WriteString(rec.Field("year"))	
	word, _,_ =strings.Cut(rec.Field("title"), " ")
	sb.WriteString(strings.ToLower(word))
		b :=byte('x')
	if rec.value != "" {
		b= rec.value[0] 	 
	}
	sb.WriteByte(b)
	sb.WriteString(rec.Field("pages")+ rec.Field("volume"))
	return sb.String()
}

// Fixkeys ensures that every record has a unique key
// contents of fldnames will be used to create a unique key
// with a,b,c etc added to ensure uniqueness; if len(fldnames)== 0 
// standard algorithm to create new citekeys. if all is true
// all keys are replaced not just duplicate records  
func FixKeys(n Node, fldnames []string, all bool) (*DedupReport, error) {
	useStd := len(fldnames) == 0
	ns := n.(*Record).children 	
	for i := 0; i < len(ns); i++ {		
		rec := ns[i].(*Record)
		if all || rec.key == "" {
			if useStd {
				rec.key = NewCiteKey(rec)
			} else {	
				rec.key = indexEntry(rec, fldnames, true)
			}	
		}
	}
	_, dr, err := Deduplicate([]Node{n}, []string{}, SetNoAction)
	if err != nil {
		return nil, err
	}
	if dr.DuplicateSetCount == 0 {
		return nil, nil
	}
	for idxTerm, nodes := range dr.DuplicateSet {
		if ndup := len(nodes); ndup > 1 {
			//TODO: may not work for dataset with many duplicates
			for i := 1; i < ndup; i++ {
				nodes[i].Node.key = idxTerm + string(rune(64+i)) //add A,B,C etc
			}
		}
	}
	return dr, nil
}

// TODO: process tex
// see https://github.com/aclements/biblib/blob/ab0e857b9198fe425ec9b02fcc293b5d9fd0c406/biblib/algo.py#L327

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
