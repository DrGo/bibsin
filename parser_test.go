package bibsin

import (
	"fmt"
	"os"
	"testing"

	"github.com/drgo/core/tu"
)

const bib1 = ` 
@string{goossens = "Goossens, Michel"}

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
func parseTestFile(t *testing.T, filename string) Node {
	t.Helper()
	n, err := Parse(nil, filename, Options{})
	tu.Equal(t, err, nil, tu.FailNow)
	tu.NotNil(t, n, tu.FailNow)
	return n
}

func TestParser(t *testing.T) {
	n := parseTestFile(t, "tests/scholar20.bib")
	rn:= n.(*Record)
	tu.Equal(t, len(rn.Children),20)
	Print(os.Stdout, n)
	//TODO: add tests 
}


func TestDedup(t *testing.T) {
	n := parseTestFile(t, "tests/scholar-dup.bib")
	 err := Deduplicate(n, DedupReport)
	tu.NotNil(t, err, tu.FailNow)
	duperr:= err.(DedupError)
	tu.Equal(t, duperr.DuplicateSetCount,3)
	fmt.Println(duperr)
}
