// server/cmd/gen-password/main.go
package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage: gen-password <plain_password>")
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(os.Args[1]),
		bcrypt.DefaultCost,
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(hash))
}
