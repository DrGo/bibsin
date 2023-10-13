package bibsin

import (
	"fmt"
	"io"
	"strings"
)

type File struct {
	Records []*Record
	name    string
}

func (f *File) AddRecord(rec *Record) {
	f.Records = append(f.Records, rec)
}

func (f *File) RecordCount() int {
	return len(f.Records)
}

func (f *File) Name() string {
	return f.name
}

func newRoot(fileName string) *File {
	return &File{name: fileName}
}

type Record struct {
	fields []Field
	key    string // citation key; ROOT for root node
	value  string // bibtex type; filenamme for root node
	line   int
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

const typBiblioTemplateBegin = `= %s
#enum(
  start: 1,
  spacing: 1.1em,
  tight: false, 
  numbering: n => text(    
    numbering("1.", %d-n+1),
  ),	
`

// report
// title={A guide to costing blood transfusion services WHO Blood Safety Unit, Geneva},
// institution={World Health Organization, Switzerland},
// pagetotal={120},
// year={1998},
// author={Mahmud S},

// inbook
// @inbook{SinghH20133,
// title={Epidemiology of Colorectal Carcinoma},
// booktitle={GI Epidemiology:Diseases and Clinical Methodology},
// edition={2},
// pages={213-221},
// pubstatus={Published},
// year={2013},
// publisher={Wiley Blackwell},
// location={United States of America},
// url={http://books.google.ca/books?hl=en\&amp;lr=\&amp;id=AEBVAgAAQBAJ\&amp;oi=fnd\&amp;pg=PA213\&amp;ots=PaTmyyC5S9\&amp;sig=TXq5xyqbpm},
// author={Singh H. and Montalban J. M. and Mahmud S.},
// editor={Mahmud S},
// keywords={book chapters},
// }

//copyrights
// @misc{Anonymous7,
// title={Drug cost tool (impute\_DPIN\_costs.sas), a macro to impute the drug costs when missing based on records that include cost information.},
// keywords={registered copyrights},
// }

// presentation
// @misc{YoungXuY20184,
// title={Analysis of the Relative Effectiveness of High-Dose Influenza Vaccines Using an Instrumental Variable Method},
// howpublished={28th European Congress of Clinical Microbiology and Infectious Diseases (ECCMID)},
// address={Spain},
// address={Madrid},
// year={2018},
// author={Young-Xu Y and Snider JT and Van Aalst R [S] and Mahmud SM and Thommes EW and Lee JKH and Greenberg DP and Chit A},
// keywords={presentations},
// }

// online
// @online{MahmudSM20178,
// title={Causal Diagrams: A Primer - Webinar},
// url={https://www.youtube.com/watch?v=ZlGAlG6uMAY},
// year={2017},
// author={Mahmud, SM},
// keywords={online resources},
// }

func AsTyp(w io.Writer, f *File, title string) (err error) {
	if _, err = fmt.Fprintf(w, typBiblioTemplateBegin, title, f.RecordCount()); err != nil {
		return err
	}
	var sb strings.Builder
	writeNotEmpty := func(s, pre, post string) bool {
		notEmpty := s != ""
		if notEmpty {
			if pre != "" {
				sb.WriteString(pre)
			}
			sb.WriteString(s)
			if post == "" {
				if s[len(s)-1] != '.' {
					sb.WriteByte('.')

				}
			} else {
				sb.WriteString(post)
			}
		}
		return notEmpty
	}
	s := ""
	for _, c := range f.Records {
		sb.Reset()
		typ := c.value
		sb.WriteByte('[') //start typst array entry
		if s = c.Field("author"); strings.HasPrefix(s, "Anonymous") {
			s = ""
		}
		writeNotEmpty(s, "", "")
		writeNotEmpty(c.Field("title"), "_", "_")
		writeNotEmpty(c.Field("year"), "* (", ")* ")
		switch typ {
		case "journal":
			writeNotEmpty(c.Field("journal"), "#underline[ ", "]. ")
			vol, issue, pages := c.Field("volume"), c.Field("issue"), c.Field("pages")
			s = strings.TrimSpace(issue)
			if s != "" {
				s = " (" + s + ") "
			}
			sb.WriteString(vol + s + pages)
			sb.WriteByte('.')
		case "report":
			writeNotEmpty(c.Field("institution"), "", "")
		case "inbook":
			writeNotEmpty(c.Field("booktitle"), "in ", "")
			writeNotEmpty(c.Field("edition"), "", " ed.")
			notEmpty := writeNotEmpty(c.Field("publisher"), "", "")
			if notEmpty {
				writeNotEmpty(c.Field("location"), "", "")
			}
		case "presentation":
			writeNotEmpty(c.Field("howpublished"), "", "")
			writeNotEmpty(c.Field("address"), "", "")
		}
		writeNotEmpty(c.Field("doi"), "", "")
		writeNotEmpty(c.Field("url"), "", "")
		sb.WriteString(".],")
		if _, err = fmt.Fprintln(w, sb.String()); err != nil {
			return nil
		}
	}
	_, err = fmt.Fprintln(w, ")")
	return err
}
