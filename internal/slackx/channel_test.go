package slackx

import "testing"

func TestParseChannelRef(t *testing.T) {
	tests := []struct {
		in, id, name string
	}{
		{"#eng", "", "eng"},
		{"eng", "", "eng"},
		{"C0123456789", "C0123456789", ""},
		{"D0123456789", "D0123456789", ""},
		{"G01234567", "G01234567", ""},
		{"  #general  ", "", "general"},
		{"", "", ""},
		{"not-an-id", "", "not-an-id"},
	}
	for _, tc := range tests {
		id, name := ParseChannelRef(tc.in)
		if id != tc.id || name != tc.name {
			t.Errorf("ParseChannelRef(%q) = (%q,%q), want (%q,%q)", tc.in, id, name, tc.id, tc.name)
		}
	}
}

func TestChannelDisplay(t *testing.T) {
	if got := (Channel{Name: "eng"}).Display(); got != "#eng" {
		t.Fatalf("got %q", got)
	}
	if got := (Channel{Name: "#eng"}).Display(); got != "#eng" {
		t.Fatalf("got %q", got)
	}
	if got := (Channel{IsIM: true, Name: "alice"}).Display(); got != "dm:alice" {
		t.Fatalf("got %q", got)
	}
	if got := (Channel{IsIM: true, ID: "D1"}).Display(); got != "dm" {
		t.Fatalf("got %q", got)
	}
	if got := (Channel{ID: "C1"}).Display(); got != "C1" {
		t.Fatalf("got %q", got)
	}
}
