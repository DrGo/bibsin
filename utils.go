package bibsin

import (
	"io"
	"os"
	"unicode"
	"unicode/utf8"
	"unsafe"
)



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

func saveWith(filename string, w func(io.Writer) error) (err error) {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		ferr := f.Close()
		if err == nil {
			err = ferr
		}
	}()
	return w(f)
}


func trimAffixes(b []byte, spacesOnly bool) string{
	start, end:= 0, len(b) 
	if end == 0 {
		return ""
	}
	found:= false 
loop:
	for ; start < end; start++ {
		switch b[start] {
		case ' ', '\t', '\n',  '\r': // consume 
		case '{', '"': //consume the outermost delimiter only 
			if spacesOnly || found {
				break loop
			}
			found= true 
		default:
			break loop
		}
	}
	end--
	found= false 
	comma := false
loop1:
	for ; end >= 0; end-- {
		switch b[end] {
		case ' ', '\t', '\n',  '\r': // consume 
		case ',': //consume the outermost comma 
			if spacesOnly || comma {
				break loop1 
			}
			comma = true 
		case '}', '"': //consume the outermost delimiter only 
			if spacesOnly || found {
				break loop1 
			}
			found= true 
		default:
			break loop1
		}
	}
	// fmt.Printf("%s: %d,%d\n", string(ob), start, end)
	return(string(b[start : end+1]))
}
