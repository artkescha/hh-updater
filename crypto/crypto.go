package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
)

// https://gist.github.com/kkirsche/e28da6754c39d5e7ea10
func Encrypt(plaintext []byte, block cipher.Block) ([]byte, []byte, error) {
	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	return aesgcm.Seal(nil, nonce, plaintext, nil), nonce, nil
}

func EncryptObj(body interface{}, block cipher.Block) (string, error) {
	var encodedCookie string
	b, err := json.Marshal(body)
	if err != nil {
		return encodedCookie, err
	}
	encryptedCookie, nonce, err := Encrypt(b, block)
	if err != nil {
		return encodedCookie, err
	}
	encryptedCookie = append(nonce, encryptedCookie...)
	return base64.StdEncoding.EncodeToString(encryptedCookie), nil
}

func Decrypt(ciphertext, nonce []byte, block cipher.Block) ([]byte, error) {
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func DecryptObj(encrypted string, block cipher.Block, v interface{}) error {
	decodedCookie, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return err
	}
	encryptedCookie := []byte(decodedCookie)
	nonce := encryptedCookie[:12]
	if len(nonce) != 12 {
		return errors.New("Nonce must be 12 characters in length")
	}
	encryptedCookie = encryptedCookie[12:]
	if len(encryptedCookie) == 0 {
		return errors.New("Encrypted Cookie missing")
	}
	data, err := Decrypt(encryptedCookie, nonce, block)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &v)
}
