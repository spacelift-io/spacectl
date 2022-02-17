package internal

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// PKCS7Unpad provides PKCS #7 unpadding mechanism.
func PKCS7Unpad(msg []byte, blockSize int) ([]byte, error) {
	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size (%d)", blockSize)
	}

	if len(msg) == 0 {
		return nil, errors.New("empty message")
	}

	if len(msg)%blockSize != 0 {
		return nil, errors.New("message is not a multiple of the block size")
	}

	return msg[:len(msg)-int(msg[len(msg)-1])], nil
}

// DecryptAES decrypts ciphertext in AES-256-CBC.
func DecryptAES(key, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext is too short")
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "could not create an AES cipher")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	cbc := cipher.NewCBCDecrypter(block, iv)
	cbc.CryptBlocks(ciphertext, ciphertext)

	plaintext, err := PKCS7Unpad(ciphertext, aes.BlockSize)
	if err != nil {
		return nil, errors.Wrap(err, "could not unpad the message")
	}

	return plaintext, nil
}

// DecryptRSA decrypts ciphertext using private key (x509 format).
func DecryptRSA(privateKey, ciphertext []byte) ([]byte, error) {
	priv, err := parsePrivateKey(privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't parse private key")
	}

	return rsa.DecryptOAEP(sha512.New(), rand.Reader, priv, ciphertext, nil)
}

// GenerateRSAKeyPair generates a key pair suitable for encryption.
func GenerateRSAKeyPair() (public []byte, private []byte, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not generate RSA key")
	}

	pubASN1, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not marshal PKIX public key")
	}

	privASN1, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't pkcs8 marshal private key")
	}

	var pubBuffer bytes.Buffer
	if err := pem.Encode(&pubBuffer, &pem.Block{Type: "RSA PUBLIC KEY", Bytes: pubASN1}); err != nil {
		return nil, nil, errors.Wrap(err, "couldn't pem-encode spacectl public key")
	}

	var privBuffer bytes.Buffer
	if err := pem.Encode(&privBuffer, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privASN1}); err != nil {
		return nil, nil, errors.Wrap(err, "couldn't pem-encode spacectl private key")
	}

	return pubBuffer.Bytes(), privBuffer.Bytes(), nil
}

func parsePrivateKey(privateKey []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, errors.New("could not decode PEM block containing private key")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		if strings.Contains(err.Error(), "ParsePKCS8PrivateKey") {
			pkcs8Priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, errors.Wrap(err, "could not parse pkcs8 encoded private key")
			}
			rsaPriv, ok := pkcs8Priv.(*rsa.PrivateKey)
			if !ok {
				return nil, errors.Errorf("expected rsa private key, got %T %+v", pkcs8Priv, pkcs8Priv)
			}
			priv = rsaPriv
		} else {
			return nil, errors.Wrap(err, "could not parse pkcs1 encoded private key")
		}
	}

	return priv, nil
}
