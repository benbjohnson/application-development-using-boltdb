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

// Ensure store can retrieve multiple users.
func TestStore_Users(t *testing.T) {
	s := OpenStore()
	defer s.Close()

	// Create some users.
	if err := s.CreateUser(&main.User{Username: "susy"}); err != nil {
		t.Fatal(err)
	} else if err := s.CreateUser(&main.User{Username: "john"}); err != nil {
		t.Fatal(err)
	}

	// Verify users can be retrieved.
	if a, err := s.Users(); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(a, []*main.User{
		{ID: 1, Username: "susy"},
		{ID: 2, Username: "john"},
	}) {
		t.Fatalf("unexpected users: %#v", a)
	}
}

// Ensure store can update a user's username.
func TestStore_SetUsername(t *testing.T) {
	s := OpenStore()
	defer s.Close()

	// Create a new user.
	if err := s.CreateUser(&main.User{Username: "susy"}); err != nil {
		t.Fatal(err)
	}

	// Update username.
	if err := s.SetUsername(1, "jimbo"); err != nil {
		t.Fatal(err)
	}

	// Verify username has changed.
	if u, err := s.User(1); err != nil {
		t.Fatal(err)
	} else if u.Username != "jimbo" {
		t.Fatalf("unexpected username: %s", u.Username)
	}
}

// Ensure store returns an error if user does not exist.
func TestStore_SetUsername_ErrUserNotFound(t *testing.T) {
	s := OpenStore()
	defer s.Close()

	// Update username.
	if err := s.SetUsername(1, "jimbo"); err != main.ErrUserNotFound {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Ensure store can remove a user.
func TestStore_DeleteUser(t *testing.T) {
	s := OpenStore()
	defer s.Close()

	// Create a new user.
	if err := s.CreateUser(&main.User{Username: "susy"}); err != nil {
		t.Fatal(err)
	}

	// Delete the user.
	if err := s.DeleteUser(1); err != nil {
		t.Fatal(err)
	}

	// Verify user does not exist.
	if u, err := s.User(1); err != nil {
		t.Fatal(err)
	} else if u != nil {
		t.Fatalf("unexpected user: %#v", u)
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
