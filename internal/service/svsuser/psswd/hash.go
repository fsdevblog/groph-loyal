package psswd

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type PasswordHash string

func (p PasswordHash) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %s", err.Error())
	}
	return string(bytes), nil
}

func (p PasswordHash) ComparePassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
