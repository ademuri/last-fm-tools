package main

import "testing"

func TestCreateUser(t *testing.T) {
	db, err := createDatabase()
	if err != nil {
		t.Fatalf("createDatabase() error: %v", err)
	}

	user := "testuser"
	err = createUser(db, user)
	if err != nil {
		t.Fatalf("createUser(%q) error: %v", user, err)
	}
}
