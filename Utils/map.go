package Utils

import "sort"

// SortMapByKey returns a sorted list of keys from a map.
// The list is ascending order
func SortMapByKey[K int32, V any](m map[K]*V) []K {
	var keys []K
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
