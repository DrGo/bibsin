package bibsin

import (
	"fmt"
	"io"
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
	return fmt.Sprintf("\n@%s{%s,\n", rec.value, rec.key)
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
