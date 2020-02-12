package main

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const randCharsAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandString(length int32) string {
	randCharBytes := make([]byte, length)
	for i := range randCharBytes {
		randCharBytes[i] = randCharsAlphabet[rand.Intn(len(randCharsAlphabet))]
	}
	return string(randCharBytes)
}
