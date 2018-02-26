package stringutils

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

func Hash(str string) (string, error) {
	password := []byte(str)
	// Hashing the password with the default cost of 10
	hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return "", errors.Wrap(err, "hash error")
	}
	return string(hashedPassword[:]), nil
}

func HashMatch(hash, str string) bool {
	hashedPassword := []byte(hash)
	password := []byte(str)
	// Comparing the password with the hash
	err := bcrypt.CompareHashAndPassword(hashedPassword, password)
	return err == nil
}
