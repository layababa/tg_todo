package main

import (
	"fmt"

	"github.com/layababa/tg_todo/server/pkg/security"
)

func main() {
	key := "a1b2c3d4e5f678901234567890abcdef1234567890abcdef1234567890abcdef"
	token := "mock-notion-token"
	encrypted, err := security.Encrypt(token, key)
	if err != nil {
		panic(err)
	}
	fmt.Println(encrypted)
}
