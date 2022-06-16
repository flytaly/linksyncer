package testutils

import (
	"reflect"
	"testing"
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
