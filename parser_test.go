package bibsin

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/drgo/core/tu"
)

func init() {
	tu.Debug = true
}

const bib1 = ` @string{goossens = "Goossens, Michel"}

This line is an implicit comment.

@article{FuMetalhalideperovskite2019,
    author = "Yongping Fu and Haiming Zhu and Jie Chen and Matthew P. Hautzinger and X.-Y. Zhu and Song Jin",
    doi = {10.1038/s41578-019-0080-9},
    journal = {Nature Reviews Materials},
    month = {feb},
    number = {3},
    pages = {169-188},
    publisher = {Springer Science and Business Media {LLC}},
    title = {Metal halide perovskite nanostructures for optoelectronic applications and the study of physical properties},
    url = {https://www.nature.com/articles/s41578-019-0080-9},
    volume = {4},
    year = {2019}
}

@comment{
    This is a comment.
    Spanning over two lines.
}

@preamble{e = mc^2}

@article{SunEnablingSiliconSolar2014,
    author = {Ke Sun and Shaohua Shen and Yongqi Liang and Paul E. Burrows and Samuel S. Mao and Deli Wang},
    doi = {10.1021/cr300459q},
    journal = {Chemical Reviews},
    month = {aug},
    number = {17},
    pages = {8662-8719},
    publisher = {American Chemical Society ({ACS})},
    title = "This title is missing a closing quote,
    url = {http://pubs.acs.org/doi/10.1021/cr300459q},
    volume = {114},
    year = {2014}
}


@string{mittelbach="Mittelbach, Franck"}

@inproceedings{LiuPhotocatalytichydrogenproduction2016,
    author = {Maochang Liu and Yubin Chen and Jinzhan Su and Jinwen Shi and Xixi Wang and Liejin Guo},
    doi = {10.1038/nenergy.2016.151},
    impactfactor = {54.000},
    journal = {Nature Energy},
    month = {sep},
    number = {11},
    pages = {16151},
    publisher = {Springer Science and Business Media {LLC}},
    title = {Photocatalytic hydrogen production using twinned nanocrystals and an unanchored {NiSx} co-catalyst},
    url = {http://www.nature.com/articles/nenergy2016151},
    volume = {1},
    year = {2016}
}


@Comment{This is another comment}
`

const bib2 = `@article{FuMetalhalideperovskite2019,
    author = "Yongping Fu and Haiming Zhu and Jie Chen and Matthew P. Hautzinger and X.-Y. Zhu and Song Jin",
    doi = {10.1038/s41578-019-0080-9},
    journal = {Nature Reviews Materials},
    month = {feb},
    number = {3},
    pages = {169-188},
    publisher = {Springer Science and Business Media {LLC}},
    title = {Metal halide perovskite nanostructures for optoelectronic applications and the study of physical properties},
    url = {https://www.nature.com/articles/s41578-019-0080-9},
    volume = {4},
    year = {2019}
}
`

func TestParser(t *testing.T) {
	f, err := Parse(strings.NewReader(bib1), "bib1", Options{})
	tu.Equal(t, err, nil, tu.FailNow)
	tu.NotNil(t, f, tu.FailNow)
	Print(os.Stdout, f)
	tu.Equal(t, len(f.Records), 3, tu.FailNow)
	c := f.Records[0]
	tu.Equal(t, c.Value(), "article")
	tu.Equal(t, c.Key(), "FuMetalhalideperovskite2019")

	tu.Equal(t, len(c.fields), 11)
	month := c.fields[3]
	tu.Equal(t, month.Key(), "month")
	tu.Equal(t, month.Value(), "feb")
	// tu.Equal(t, c.Field("pages"), "16151", tu.FailNow)
}

func parseTestFile(t *testing.T, filename string) *File {
	t.Helper()
	n, err := Parse(nil, filename, Options{})
	tu.Equal(t, err, nil, tu.FailNow)
	tu.NotNil(t, n, tu.FailNow)
	return n
}

func TestParseFile(t *testing.T) {
	n := parseTestFile(t, "tests/scholar20.bib")
	tu.Equal(t, n.RecordCount(), 20)
	Print(os.Stdout, n)
	//TODO: add tests
}

// func TestDedup(t *testing.T) {
// 	n := parseTestFile(t, "tests/scholar-dup.bib")
// 	err := Deduplicate(n, DedupReport)
// 	tu.NotNil(t, err, tu.FailNow)
// 	dr:= err.(DedupError)
// 	tu.Equal(t, dr.DuplicateSetCount,3)
// 	fmt.Println(dr)
// }

func TestOnlyASCIIAlphaNumeric(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"[test   name	\n", "testname"},
		{"[test123   :Name	\n", "test123name"},
		{"", ""},
		{"  ", ""},
	}

	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			tu.Equal(t, onlyASCIAlphaNumeric(test.in), test.out)
		})
	}
}

func TestDedupByKey(t *testing.T) {
	n1 := parseTestFile(t, "tests/scholar-dup.bib")
	_, dr, err := Deduplicate([]*File{n1}, []string{}, SetNoAction)
	tu.NotNil(t, dr, tu.FailNow)
	tu.Equal(t, err, nil)
	tu.Equal(t, dr.DuplicateSetCount, 3)
	fmt.Println(dr)
}

func TestDedupByContent(t *testing.T) {
	n1 := parseTestFile(t, "tests/scholar-dup.bib")
	_, dr, err := Deduplicate([]*File{n1}, []string{"year", "journal"}, SetNoAction)
	tu.NotNil(t, dr, tu.FailNow)
	tu.Equal(t, err, nil)
	tu.Equal(t, dr.DuplicateSetCount, 3)
	fmt.Println(dr)

	n2 := parseTestFile(t, "tests/scholar20.bib")
	_, dr, err = Deduplicate([]*File{n1, n2}, []string{"year", "journal"}, SetNoAction)
	tu.NotNil(t, dr, tu.FailNow)
	tu.Equal(t, err, nil)
	tu.Equal(t, dr.DuplicateSetCount, 3)
	fmt.Println(dr)

	res, dr, err := Deduplicate([]*File{n1, n2}, []string{"year", "journal"}, SetIntersect)
	tu.Equal(t, err, nil)
	tu.NotNil(t, dr, tu.FailNow)
	tu.NotNil(t, res, tu.FailNow)
	tu.Equal(t, len(res.Records), 20)

	res, dr, err = Deduplicate([]*File{n1, n2}, []string{"year", "journal"}, SetUnion)
	tu.Equal(t, err, nil)
	tu.NotNil(t, dr, tu.FailNow)
	tu.NotNil(t, res, tu.FailNow)
	tu.Equal(t, len(res.Records), 20)

}

func TestDedupBib(t *testing.T) {
	pr := func(f *File, err error, expect int) {
		if err != nil {
			panic(err)
		}
		if f == nil {
			return
		}
		tu.P("%s %d records found\n", f.name, f.RecordCount())
		tu.Equal(t, f.RecordCount(), expect)
	}
	bib1, err := Parse(strings.NewReader(bib1), "bib1", Options{})
	pr(bib1, err, 3)
	bib2, err := Parse(strings.NewReader(bib2), "bib2", Options{})
	pr(bib2, err, 1)
	res, dr, err := Deduplicate([]*File{bib2, bib1}, []string{"year", "title"}, SetNoAction)
	pr(res, err, -1)
	// fmt.Println(dr)
	err = saveWith("./tests/dedup.txt", func(w io.Writer) (err error) {
		return dr.Print(w)
	})
	pr(res, err, -1)

	f, dr, err := Deduplicate([]*File{bib2, bib1}, []string{"year", "title"}, SetIntersect)
	pr(f, err, 1)
	tu.P("%d records processed\n", dr.ResultSetCount)
	tu.Equal(t, dr.ResultSetCount, 1)

	err = saveWith("./tests/bibmerged.bib", func(w io.Writer) error {
		f, dr, err := Deduplicate([]*File{bib2, bib1}, []string{"year", "title"}, SetUnion)
		pr(f, err, 3)
		tu.P("%d records processed\n", dr.ResultSetCount)
		tu.Equal(t, dr.ResultSetCount, 3)
		return Print(w, f)
	})

	if err != nil {
		panic(err)
	}

	// Output: 413 records found
}
func ExampleDeduplicateByContents() {
	ccv, err := Parse(nil, "./tests/salah.bib", Options{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d records found\n", len(ccv.Records))
	scholar, err := Parse(nil, "./tests/scholar.bib", Options{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d records found\n", len(scholar.Records))
	_, dr, err := Deduplicate([]*File{ccv, scholar}, []string{"year", "title"}, SetNoAction)
	if err != nil {
		panic(err)
	}
	// fmt.Printf("%d records found\n", len(res.Records))
	// fmt.Println(dr)
	err = saveWith("./tests/dedup.txt", func(w io.Writer) (err error) {
		return dr.Print(w)
	})
	if err != nil {
		panic(err)
	}

	err = saveWith("./tests/merged.bib", func(w io.Writer) error {
		n, dr, err := Deduplicate([]*File{ccv, scholar}, []string{"year", "title"}, SetUnion)
		if err != nil {
			return err
		}
		fmt.Printf("%d records processed\n", dr.ResultSetCount)
		return Print(w, n)
	})

	if err != nil {
		panic(err)
	}

	// Output: 413 records found
}

func TestSort(t *testing.T) {
	n1 := parseTestFile(t, "tests/merged.bib")
	err := saveWith("./tests/sorted.bib", func(w io.Writer) error {
		err := Sort(n1, "type,-year")
		tu.Equal(t, err, nil)
		return Print(w, n1)
	})

	tu.Equal(t, err, nil)
}

func TestFixKeys(t *testing.T) {
	n1 := parseTestFile(t, "tests/sorted.bib")
	err := saveWith("./tests/fixdup.bib", func(w io.Writer) error {
		dr, err := FixKeys(n1, nil, false) // all=false: only generate keys for missing keys
		tu.Equal(t, err, nil, tu.FailNow)
		// tu.PL(err)
		Print(w, n1)
		dr.Print(os.Stdout)
		return nil // Print(w, n1)
	})
	tu.Equal(t, err, nil)
	tu.Equal(t, ValidKeys(n1), true)

}

func TestTrimAffixes(t *testing.T) {
	tests := []struct {
		in         string
		out        string
		spacesOnly bool
	}{
		{`"test name"`, `test name`, false},
		{` {"test1"}`, `"test1"`, false},
		{` {"test2"},`, `"test2"`, false},
		{` {"test3"}`, `"test3"`, false},
		{`FuMetalhalideperovskite2019,`, `FuMetalhalideperovskite2019`, false},
		{`{}`, ``, false},
		{`{},`, ``, false},
		{`"  {}""`, `{}"`, false},
		{``, ``, false},
		{` {"spaces"}	`, `{"spaces"}`, true},
	}
	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			s := []byte(test.in)
			tu.Equal(t, trimAffixes(s, test.spacesOnly), test.out)
		})
	}
}

func TestSplit(t *testing.T) {
	srcf := parseTestFile(t, "tests/fixdup.bib")
	files := Split(srcf) 
	tu.Equal(t, len(files), 6)
	for name, f := range files {
		for i := 0; i < 2; i++ {
			_, dr, err := Deduplicate([]*File{f}, []string{"year", "title"}, SetUnion)
			if err != nil {
				panic(err)
			}
			tu.PL(name, dr.DuplicateSetCount, "\n")
		}
		saveWith(filepath.Join("./tests/sub", name+".bib"), func(w io.Writer) error {
			Print(w, f)
			return nil

		})
	}
	// tu.Equal(t, err, nil)
	// tu.Equal(t, ValidKeys(f), true)

}
