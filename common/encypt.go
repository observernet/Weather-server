package common

import (
    "bytes"
	"crypto/hmac"
    "crypto/sha256"
    "crypto/aes"
    "crypto/cipher"
    "encoding/base64"
)

func EncyptData(data string, key string) string {

	h := hmac.New(sha256.New, []byte(key))
    h.Write([]byte(data))
    return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func CompressKeyData(plaintext string, key string, iv string) string {

    bKey := []byte(key)
	bIV := []byte(iv)
	bPlaintext := PKCS5Padding([]byte(plaintext), len(plaintext))
	block, _ := aes.NewCipher(bKey)
	ciphertext := make([]byte, len(bPlaintext))
	mode := cipher.NewCBCEncrypter(block, bIV)
	mode.CryptBlocks(ciphertext, bPlaintext)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func PKCS5Padding(ciphertext []byte, after int) []byte {
	padding := (aes.BlockSize - len(ciphertext)%aes.BlockSize)
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}
