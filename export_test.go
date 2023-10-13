package bibsin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/drgo/core/tu"
)

const testPath = "./tests"

func TestExport(t *testing.T) {
	scholar := parseTestFile(t, filepath.Join(testPath, "scholar.bib"))
	ccv := parseTestFile(t, filepath.Join(testPath, "salah.bib"))

	merged, dr, err := Deduplicate([]*File{ccv, scholar}, []string{"year", "title"}, SetUnion)
	dr.Print(os.Stdout)
	tu.Equal(t, err, nil, tu.FailNow)
	err = Sort(merged, "type,-year")
	tu.Equal(t, err, nil)

	_, err = FixKeys(merged, nil, false) // all=false: only generate keys for missing keys
	tu.Equal(t, err, nil)

	err = FixTypes(merged)
	tu.Equal(t, err, nil)

	files := Split(merged)
	tu.Equal(t, len(files), 6)
	typDir := filepath.Join(testPath, "typ")
	EnsureEmptyDir(typDir)
	for _, f := range files {
		err = ExportTyp(f, typDir)
		tu.Equal(t, err, nil, tu.FailNow)
	}
}

func EnsureEmptyDir(dirName string) error {
	err := os.RemoveAll(dirName)
	if err != nil {
		return err
	}
	// dir does not exist or was deleted
	return os.Mkdir(dirName, 0750)
}

func EnsureDir(dirName string) error {
	err := os.Mkdir(dirName, 0750)
	if err == nil {
		return nil
	}
	if os.IsExist(err) {
		info, err := os.Stat(dirName)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return errors.New("path exists but is not a directory")
		}
		return nil
	}
	return err
}

func TestFixType(t *testing.T) {
	f := newRoot("test")
	f.AddRecord(&Record{value: "misc", fields: []Field{Field{
		key: "keywords", value: "registered copyrights",
	}}})
	err := FixTypes(f)
	tu.Equal(t, err, nil)
	Print(os.Stdout, f)
}
