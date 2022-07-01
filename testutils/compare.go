package testutils

import (
	"reflect"
	"testing"

	"golang.org/x/exp/constraints"
)

func Compare(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
		t.Errorf("difference %+v", Difference(got, want))
	}
}

// Difference between two slices
func Difference(slice1, slice2 []string) []string {
	diff := []string{}
	m := map[string]int{}

	for _, v := range slice1 {
		m[v] = 1
	}
	for _, v := range slice2 {
		m[v] = m[v] + 1
	}

	for k, v := range m {
		if v == 1 {
			diff = append(diff, k)
		}
	}

	return diff
}

func min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func StringDifference(s1, s2 string) (r1, r2 string) {
	for pos := range s1 {
		if s1[pos] != s2[pos] {
			return s1[max(0, pos-10):min(pos+10, len(s1))], s2[max(0, pos-10):min(pos+10, len(s2))]
		}
	}
	return "", ""
}
