package crypt

import (
	"crypto/rand"
	"fmt"
	"math"
	"reflect"
	"testing"
)

func getBytes(count int) []byte {
	v := make([]byte, count)
	if got, err := rand.Read(v); got != count || err != nil {
		panic(fmt.Sprintf("Could not read %d bytes, got only %d and error %v", count, got, err))
	}
	return v
}

func TestEncryptDecrypt(t *testing.T) {
	var tests []int
	for i := 1; i < 128; i++ {
		tests = append(tests, i)
	}
	for i := 8; i < 18; i++ {
		tests = append(tests, int(math.Pow(2, float64(i))))
	}
	e, err := New("this is a passphrase")
	if err != nil {
		t.Fatalf("could not initialize encryption: %v", err)
	}
	for _, count := range tests {
		src := getBytes(count)
		var dst []byte
		if dst, err = e.Encrypt(src); err != nil {
			t.Errorf("encrypting %d bytes: %v", count, err)
		}
		var dsrc []byte
		if dsrc, err = e.Decrypt(dst); err != nil {
			t.Errorf("decrypting %d bytes: %v", count, err)
		}
		if !reflect.DeepEqual(src, dsrc) {
			t.Errorf("decryption mismatch for %d bytes:\n% x\n% x", count, src, dsrc)
		}
	}
}
