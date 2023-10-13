package bibsin

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// ExportTyp takes a deduplicated bib *File with fixed keys and types
// and outputs several typ-formatted files ready for typesetting
func ExportTyp(bib *File, outDirName string) error {
	files := Split(bib)
	if len(files) == 0 {
		return fmt.Errorf("nothign to export")
	}
	for name, f := range files {
		_, _, err := Deduplicate([]*File{f}, []string{"year", "title"}, SetUnion)
		if err != nil {
			return err
		}
	
		secName := ""
		switch name {
		case "article":
			secName= "Referred Articles"
		// case "report":
		// 	name = "Reports"
		case "inbook":
			secName = "Book Chapters"
		case "online":
			secName = "Software and Online Resources"
		default:
			secName = strings.Title(name)
		}
		saveWith(filepath.Join(outDirName, name+".typ"), func(w io.Writer) error {
			return AsTyp(w, f, secName)
		})
	}
	return nil
}
