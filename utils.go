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
