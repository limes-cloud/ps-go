package rsa

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"ps-go/errors"
)

// 私钥生成
//openssl genrsa -out rsa_private_key.pem 1024
// 公钥: 根据私钥生成
//openssl rsa -in rsa_private_key.pem -pubout -out rsa_public_key.pem

func EncryptToBase64(orig string, key []byte) (string, error) {
	data, err := Encrypt([]byte(orig), key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func EncryptToHex(orig string, key []byte) (string, error) {
	data, err := Encrypt([]byte(orig), key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func DecryptFromBase64(orig string, key []byte) (string, error) {
	cryptByte, err := base64.StdEncoding.DecodeString(orig)
	if err != nil {
		return "", err
	}
	resp, err := Decrypt(cryptByte, key)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

func DecryptFromHex(orig string, key []byte) (string, error) {
	cryptByte, err := hex.DecodeString(orig)
	if err != nil {
		return "", err
	}
	resp, err := Decrypt(cryptByte, key)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

// Encrypt 公钥加密
func Encrypt(origData []byte, key []byte) ([]byte, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, errors.New("public key error")
	}
	// 解析公钥
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	// 类型断言
	pub := pubInterface.(*rsa.PublicKey)
	//加密
	return rsa.EncryptPKCS1v15(rand.Reader, pub, origData)
}

// Decrypt 私钥解密
func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	//解密
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, errors.New("private key error")
	}
	//解析PKCS1格式的私钥
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	// 解密
	return rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
}
