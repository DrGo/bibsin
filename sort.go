package bibsin

import (
	"fmt"
	"sort"
	"strconv"
)

const (
	Missing = 1<<32 - 1
)

func Sort(f *File, flds string) error {
	if f.RecordCount() == 0 {
		return fmt.Errorf("nothing to sort")
	}
	// special common case case
	if flds == "type,-year" {
		recs := f.Records
		sort.Slice(recs, func(i, j int) bool {
			ni, nj := recs[i], recs[j]
			if ni.value != nj.value {
				return ni.value < nj.value //record type
			}
			yi, err := strconv.Atoi(ni.Field("year"))
			if err != nil {
				yi = Missing
			}
			yj, err := strconv.Atoi(nj.Field("year"))
			if err != nil {
				yj = Missing
			}
			return yi > yj // descending sort
		})
		return nil
	}
	return fmt.Errorf("not implemented")
}
