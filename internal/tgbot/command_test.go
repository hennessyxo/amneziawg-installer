package tgbot

import (
	"reflect"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		text    string
		wantCmd Command
		wantOK  bool
	}{
		{"/new phone", Command{Name: "new", Args: []string{"phone"}}, true},
		{"  /list  ", Command{Name: "list", Args: []string{}}, true},
		{"/revoke@AwgBot laptop", Command{Name: "revoke", Args: []string{"laptop"}}, true},
		{"/AUTH s3cret", Command{Name: "auth", Args: []string{"s3cret"}}, true},
		{"hello there", Command{}, false},
		{"", Command{}, false},
	}
	for _, tc := range tests {
		got, ok := ParseCommand(tc.text)
		if ok != tc.wantOK {
			t.Errorf("ParseCommand(%q) ok=%v, want %v", tc.text, ok, tc.wantOK)
			continue
		}
		if ok && !reflect.DeepEqual(got, tc.wantCmd) {
			t.Errorf("ParseCommand(%q) = %+v, want %+v", tc.text, got, tc.wantCmd)
		}
	}
}
