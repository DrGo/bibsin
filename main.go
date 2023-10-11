package bibsin

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"
)

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
	Parent *File
}

type DedupMap = map[string][]NodeInfo

type DedupReport struct {
	DuplicateSetCount int
	DuplicateSet      DedupMap
	ResultSetCount    int
}

func (dr *DedupReport) Print(w io.Writer) (err error) {
	if dr == nil || dr.DuplicateSetCount == 0 {
		return nil
	}
	fmt.Fprintf(w, "%d duplicate sets found\n", dr.DuplicateSetCount)
	for idxTerm, nodes := range dr.DuplicateSet { //
		if ndup := len(nodes); ndup > 1 {
			_, err = fmt.Fprintf(w, "%s\n[%s] has %d occurrences in lines \n", strings.Repeat("*", 60), idxTerm, ndup)
			for _, n := range nodes {
				// write filename: line
				_, err = fmt.Fprintf(w, "%s:%d\n", n.Parent.Name(), n.Node.Line())
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
	if err := dr.Print(b); err != nil {
		b.WriteString("error: " + err.Error())
	}
	return b.String()
}

//	func Query(nodes []Node, query string)(Node, error) {
//		for _, r := range nodes {
//			for _, c := range r.Children() {
//	}
//
// indexEntry returns a string concating values of fields
func indexEntry(rec *Record, fldNames []string, raw bool) string {
	var sb strings.Builder
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
func Deduplicate(files []*File, fldNames []string, action SetActionType) (*File, *DedupReport, error) {
	if len(files)*files[0].RecordCount() == 0 {
		return nil, nil, fmt.Errorf("nothing to deduplicate")
	}
	hasFields := len(fldNames) > 0
	citekey := !hasFields || slices.Contains(fldNames, "citekey")
	// print("citekey"); print(citekey)
	dupSet := make(DedupMap, files[0].RecordCount()*len(files))
	for _, r := range files {
		for _, c := range r.Records {
			idx := ""
			if hasFields {
				idx = indexEntry(c, fldNames, false)
			}
			if citekey {
				idx = idx + c.Key()
			}
			dupSet[idx] = append(dupSet[idx], NodeInfo{c, r})
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
		for _, recs := range dupSet {
			if ndup := len(recs); ndup > 1 { //duplicates
				res.AddRecord(recs[0].Node) //print the first in the set
				dr.ResultSetCount++
			}
		}
		return res, dr, nil
	}
	if action == SetUnion {
		res := newRoot("union.bib")
		for _, recs := range dupSet {
			res.AddRecord(recs[0].Node)
			dr.ResultSetCount++
		}
		return res, dr, nil
	}
	return nil, nil, fmt.Errorf("invalid set action")
}

// ValidKeys checks if all records have citekeys and all are unique
func ValidKeys(n *File) bool {
	_, dr, err := Deduplicate([]*File{n}, []string{}, SetNoAction)
	if err != nil {
		return true // only error is nothign to duplicate
	}
	return dr.DuplicateSetCount == 0
}

// ScholarCiteKey generates a new key using last name of the first author + pub year+
// first word of the title + first letter of article type + page or volume #
func NewCiteKey(rec *Record) string {
	var sb strings.Builder
	word, _, found := strings.Cut(rec.Field("author"), ",")
	if !found {
		word, _, _ = strings.Cut(rec.Field("author"), " ")
	}
	sb.WriteString(strings.ToLower(word))
	sb.WriteString(rec.Field("year"))
	word, _, _ = strings.Cut(rec.Field("title"), " ")
	sb.WriteString(strings.ToLower(word))
	b := byte('x')
	if rec.value != "" {
		b = rec.value[0]
	}
	sb.WriteByte(b)
	sb.WriteString(rec.Field("pages") + rec.Field("volume"))
	return sb.String()
}

// Fixkeys ensures that every record has a unique key
// contents of fldnames will be used to create a unique key
// with a,b,c etc added to ensure uniqueness; if len(fldnames)== 0
// standard algorithm to create new citekeys. if all is true
// all keys are replaced not just duplicate records
func FixKeys(f *File, fldnames []string, all bool) (*DedupReport, error) {
	useStd := len(fldnames) == 0
	for _, rec:= range f.Records { 
		if all || rec.key == "" {
			if useStd {
				rec.key = NewCiteKey(rec)
			} else {
				rec.key = indexEntry(rec, fldnames, false)
			}
		}
	}
	// dedup in terms of citykey
	_, dr, err := Deduplicate([]*File{f}, []string{}, SetNoAction)
	if err != nil {
		return nil, err
	}
	if dr.DuplicateSetCount == 0 {
		return nil, nil
	}
	for _, nodes := range dr.DuplicateSet {
		if ndup := len(nodes); ndup > 1 {
			//TODO: may not work for dataset with many duplicates
			for i := 1; i < ndup; i++ {
				nodes[i].Node.key = nodes[i].Node.key + string(rune(64+i)) //add A,B,C etc
			}
		}
	}
	return dr, nil
}

// Split splits a set into a separate set for each citation type
func Split(f *File) map[string]*File {
	res := make(map[string]*File, 10)
	for _, rec := range f.Records {
		sub, ok := res[rec.Value()]
		if !ok {
			sub = newRoot(rec.Value())
			res[rec.Value()]= sub
		}
		sub.AddRecord(rec)
	}
	return res
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
