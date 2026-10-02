package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

// hashgen mencetak hash bcrypt untuk setiap argumen yang diberikan.
// Dipakai untuk menyiapkan password awal pada berkas seeder, karena
// hash tidak mungkin ditulis manual.
//
// Pemakaian: go run ./cmd/hashgen teks1 teks2 ...
func main() {
	for _, plain := range os.Args[1:] {
		hash, err := bcrypt.GenerateFromPassword([]byte(plain), 12)
		if err != nil {
			fmt.Fprintln(os.Stderr, "gagal hash:", err)
			os.Exit(1)
		}
		fmt.Println(string(hash))
	}
}
