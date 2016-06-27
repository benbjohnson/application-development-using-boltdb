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

	// Initialize buckets to guarantee that they exist.
	if err := s.db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("Users"))

		return nil
	}); err != nil {
		db.Close()
		return err
	}

	return nil
}

// Close shuts down the store.
func (s *Store) Close() error {
	return s.db.Close()
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
	b := tx.Bucket([]byte("Users"))

	// The sequence is an autoincrementing integer that is transactionally safe.
	seq, _ := b.NextSequence()
	u.ID = int(seq)

	// Marshal our user into bytes.
	buf, err := u.MarshalBinary()
	if err != nil {
		return err
	}

	// Save user to the bucket.
	if err := b.Put(itob(u.ID), buf); err != nil {
		return err
	}

	// Commit transaction and exit.
	return tx.Commit()
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
	buf := tx.Bucket([]byte("Users")).Get(itob(id))
	if buf == nil {
		return nil, nil
	}

	// Unmarshal bytes into a user.
	var u User
	if err := u.UnmarshalBinary(buf); err != nil {
		return nil, err
	}

	return &u, nil
}

// itob encodes v as a big endian integer.
func itob(v int) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	return buf
}
