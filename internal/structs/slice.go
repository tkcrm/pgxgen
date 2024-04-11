package structs

import (
	"fmt"
	"slices"
	"sort"
)

type StructSlice []*StructParameters

func (st *StructSlice) ExistStructIndex(name string) (int, *StructParameters) {
	for index, s := range *st {
		if s.Name == name {
			return index, s
		}
	}
	return -1, nil
}

func (st *StructSlice) Sort(priorityNames ...string) error {
	if len(priorityNames) == 0 {
		return nil
	}

	names := make([]string, 0, len(*st))
	for _, name := range priorityNames {
		existStructIndex, _ := st.ExistStructIndex(name)
		if existStructIndex == -1 {
			return fmt.Errorf("sort error: undefined struct %s", name)
		}

		names = append(names, name)
	}

	notPriorityNames := make([]string, 0, len(*st)-len(priorityNames))
	for _, v := range *st {
		if slices.Contains(names, v.Name) {
			continue
		}
		notPriorityNames = append(notPriorityNames, v.Name)
	}
	sort.Strings(notPriorityNames)

	names = append(names, notPriorityNames...)

	sorted := make(StructSlice, 0, len(names))
	for _, n := range names {
		for _, s := range *st {
			if s.Name == n {
				sorted = append(sorted, s)
			}
		}
	}

	*st = sorted

	return nil
}

func ConvertStructsToSlice(st Structs) StructSlice {
	res := make(StructSlice, 0, len(st))

	for _, s := range st {
		res = append(res, s)
	}

	return res
}
