package bibsin

import (
	"fmt"
	"sort"
	"strconv"
)

const (
	Missing = 1<<32 - 1
)

func Sort(root Node, flds string) error {
	if !root.IsRoot() || len(root.Children()) == 0 {
		return fmt.Errorf("nothing to sort")
	}
	// special common case case  
	if flds == "type,-year" {
		recs := root.Children()
		sort.Slice(recs, func(i, j int) bool {
			ni, nj := recs[i].(*Record), recs[j].(*Record)
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
