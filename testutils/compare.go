package testutils

import (
	"testing"

	"golang.org/x/exp/constraints"
)

func Compare(t *testing.T, got, want []string) {
	t.Helper()
	d := Difference(got, want)
	if len(d) > 0 {
		t.Errorf("got %+v, want %+v", got, want)
		t.Errorf("difference %+v", Difference(got, want))
	}
}

func CompareMapKeys[V any](t *testing.T, got map[string]V, wantKeys []string) {
	diff := []string{}

	gotKeys := make([]string, 0, len(got))
	for k := range got {
		gotKeys = append(gotKeys, k)
	}

	wantMap := map[string]bool{}
	for _, v := range wantKeys {
		wantMap[v] = true
	}

	if len(gotKeys) > len(wantKeys) {
		for _, v := range gotKeys {
			if !wantMap[v] {
				diff = append(diff, v)
			}
		}
		t.Errorf("got %+v, want %+v", gotKeys, wantKeys)
		t.Errorf("excessive elements: %+v", diff)
		return
	}

	for _, v := range wantKeys {
		if _, ok := got[v]; !ok {
			diff = append(diff, v)
		}
	}
	if len(diff) > 0 {
		t.Errorf("got %+v, want %+v", got, wantKeys)
		t.Errorf("missing %+v", diff)
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
