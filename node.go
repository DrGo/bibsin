package bibsin

import (
	"fmt"
	"io"
)

type File struct {
	Records []*Record
	name string 
}

func (f *File) AddRecord(rec *Record){
	f.Records=append(f.Records, rec)
}


func (f *File) RecordCount()int{
	return len(f.Records)
}

func (f *File) Name()string{
	return f.name
}

func newRoot(fileName string) *File {
	return &File{name: fileName}
}

type Record struct {
	fields []Field
	key      string // citation key; ROOT for root node
	value    string // bibtex type; filenamme for root node 
	line     int
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
	return fmt.Sprintf("\n@%s{%s,\n", rec.value, rec.key)
}

// func (n *Record) Children() []Node {
// 	return n.children
// }

func (n *Record) addField(c Field) {
	n.fields = append(n.fields, c)
}

func (n *Record) Field(fieldName string) string {
	for _, fld := range n.fields {
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


func Print(w io.Writer, n any) error {
	//FIXME: check for errors
	switch n := n.(type) {
	case *File:
			for _, c := range n.Records {
				Print(w, c)
			}
			return nil
	case *Record:		
		fmt.Fprintf(w, n.BibtexRepr())
		for i, c := range n.fields {
			Print(w, c)
			if i < len(n.fields) {
				fmt.Fprintln(w, ",")
			}
		}
		fmt.Fprintln(w, "}")
	case Field:
		fmt.Fprintf(w, n.BibtexRepr())
	default:
		return fmt.Errorf("Unknown Node type")
	}
	return nil
}
