package main

import (
	"encoding/binary"

	"github.com/benbjohnson/application-development-using-boltdb/internal"
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
)

//go:generate protoc --gogo_out=. internal/internal.proto

// User represents a user in our system.
type User struct {
	ID       int
	Username string
}

// MarshalBinary encodes a user to binary format.
func (u *User) MarshalBinary() ([]byte, error) {
	return proto.Marshal(&internal.User{
		ID:       proto.Int64(int64(u.ID)),
		Username: proto.String(u.Username),
	})
}

// UnmarshalBinary decodes a user from binary data.
func (u *User) UnmarshalBinary(data []byte) error {
	var pb internal.User
	if err := proto.Unmarshal(data, &pb); err != nil {
		return err
	}

	u.ID = int(pb.GetID())
	u.Username = pb.GetUsername()

	return nil
}

// Store represents the data storage layer.
type Store struct {
	// Filepath to the data file.
	Path string

	db *bolt.DB
}

// Open opens and initializes the store.
func (s *Store) Open() error {
	// Open bolt database.
	db, err := bolt.Open(s.Path, 0666, nil)
	if err != nil {
		return err
	}
	s.db = db

	// Start a writable transaction.
	tx, err := s.db.Begin(true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Initialize buckets to guarantee that they exist.
	tx.CreateBucketIfNotExists([]byte("Users"))

	// Commit the transaction.
	return tx.Commit()
}

// Close shuts down the store.
func (s *Store) Close() error {
	return s.db.Close()
}

// User retrieves a user by ID.
func (s *Store) User(id int) (*User, error) {
	// Start a readable transaction.
	tx, err := s.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Read encoded user bytes.
	v := tx.Bucket([]byte("Users")).Get(itob(id))
	if v == nil {
		return nil, nil
	}

	// Unmarshal bytes into a user.
	var u User
	if err := u.UnmarshalBinary(v); err != nil {
		return nil, err
	}

	return &u, nil
}

// Users retrieves a list of all users.
func (s *Store) Users() ([]*User, error) {
	// Start a readable transaction.
	tx, err := s.db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Create a cursor on the user's bucket.
	c := tx.Bucket([]byte("Users")).Cursor()

	// Read all users into a slice.
	var a []*User
	for k, v := c.First(); k != nil; k, v = c.Next() {
		var u User
		if err := u.UnmarshalBinary(v); err != nil {
			return nil, err
		}
		a = append(a, &u)
	}

	return a, nil
}

// CreateUser creates a new user in the store.
// The user's ID is set to u.ID on success.
func (s *Store) CreateUser(u *User) error {
	// Start a writeable transaction.
	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Retrieve bucket.
	bkt := tx.Bucket([]byte("Users"))

	// The sequence is an autoincrementing integer that is transactionally safe.
	seq, _ := bkt.NextSequence()
	u.ID = int(seq)

	// Marshal our user into bytes.
	buf, err := u.MarshalBinary()
	if err != nil {
		return err
	}

	// Save user to the bucket.
	if err := bkt.Put(itob(u.ID), buf); err != nil {
		return err
	}

	// Commit transaction and exit.
	return tx.Commit()
}

// SetUsername updates the username for a user.
func (s *Store) SetUsername(id int, username string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte("Users"))

		// Retrieve encoded user and decode.
		var u User
		if v := bkt.Get(itob(id)); v == nil {
			return ErrUserNotFound
		} else if err := u.UnmarshalBinary(v); err != nil {
			return err
		}

		// Update user.
		u.Username = username

		// Encode and save user.
		if buf, err := u.MarshalBinary(); err != nil {
			return err
		} else if err := bkt.Put(itob(id), buf); err != nil {
			return err
		}

		return nil
	})
}

// DeleteUser removes a user by id.
func (s *Store) DeleteUser(id int) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("Users")).Delete(itob(id))
	})
}

// itob encodes v as a big endian integer.
func itob(v int) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	return buf
}

// User related errors.
var (
	ErrUserNotFound = Error("user not found")
)

// Error represents an application error.
type Error string

func (e Error) Error() string { return string(e) }
