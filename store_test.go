package main_test

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	main "github.com/benbjohnson/application-development-using-boltdb"
)

// Ensure store can create a new user.
func TestStore_CreateUser(t *testing.T) {
	s := OpenStore()
	defer s.Close()

	// Create a new user.
	u := &main.User{Username: "susy"}
	if err := s.CreateUser(u); err != nil {
		t.Fatal(err)
	} else if u.ID != 1 {
		t.Fatalf("unexpected ID: %d", u.ID)
	}

	// Verify user can be retrieved.
	other, err := s.User(1)
	if err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(other, u) {
		t.Fatalf("unexpected user: %#v", other)
	}
}

// Store is a test wrapper for main.Store.
type Store struct {
	*main.Store
}

// NewStore returns a new instance of Store in a temporary path.
func NewStore() *Store {
	f, err := ioutil.TempFile("", "appdevbolt-")
	if err != nil {
		panic(err)
	}
	f.Close()

	return &Store{
		Store: &main.Store{
			Path: f.Name(),
		},
	}
}

// OpenStore returns a new, open instance of Store.
func OpenStore() *Store {
	s := NewStore()
	if err := s.Open(); err != nil {
		panic(err)
	}
	return s
}

// Close closes the store and removes the underlying data file.
func (s *Store) Close() error {
	defer os.Remove(s.Path)
	return s.Store.Close()
}
