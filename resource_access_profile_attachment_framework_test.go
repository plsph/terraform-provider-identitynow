package main

import (
	"reflect"
	"testing"
)

func TestAccessProfilesToDetach(t *testing.T) {
	got := accessProfilesToDetach([]string{"a", "b", "c"}, []string{"b", "c", "unmanaged"})
	if !reflect.DeepEqual(got, []string{"b", "c"}) {
		t.Fatalf("expected only managed and attached profiles, got %#v", got)
	}
	if got := accessProfilesToDetach([]string{"a"}, nil); len(got) != 0 {
		t.Fatalf("expected nothing to detach, got %#v", got)
	}
}

func TestOrderByPriorIDs(t *testing.T) {
	got := orderByPriorIDs([]string{"c", "a", "new", "b"}, []string{"a", "b", "c", "removed"}, func(id string) string { return id })
	if !reflect.DeepEqual(got, []string{"a", "b", "c", "new"}) {
		t.Fatalf("unexpected order %#v", got)
	}
}
