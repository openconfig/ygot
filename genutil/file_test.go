package genutil

import (
	"os"
	"testing"
)

func TestOpenSyncFile(t *testing.T) {
	dir, err := os.MkdirTemp(".", "ygot-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	filename := dir + "/foo.txt"
	file := OpenFile(filename)
	file.WriteString("42")
	SyncFile(file)

	bytes, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(bytes), "42"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
