package aes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
)

func EncryptToBase64(orig string, key string) (string, error) {
	data, err := Encrypt(orig, key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func EncryptToHex(orig string, key string) (string, error) {
	data, err := Encrypt(orig, key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func DecryptFromBase64(orig string, key string) (string, error) {
	cryptByte, err := base64.StdEncoding.DecodeString(orig)
	if err != nil {
		return "", err
	}
	return Decrypt(cryptByte, key)
}

func DecryptFromHex(orig string, key string) (string, error) {
	cryptByte, err := hex.DecodeString(orig)
	if err != nil {
		return "", err
	}
	return Decrypt(cryptByte, key)
}

func Encrypt(orig string, key string) ([]byte, error) {
	// 转成字节数组
	origData := []byte(orig)
	k := []byte(key)

	// 分组秘钥
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 补全码
	origData = PKCS7Padding(origData, blockSize)
	// 加密模式
	blockMode := cipher.NewCBCEncrypter(block, k[:blockSize])
	// 创建数组
	crypt := make([]byte, len(origData))
	// 加密
	blockMode.CryptBlocks(crypt, origData)
	return crypt, nil
}

func Decrypt(crypt []byte, key string) (string, error) {
	// 转成字节数组
	k := []byte(key)

	// 分组秘钥
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 加密模式
	blockMode := cipher.NewCBCDecrypter(block, k[:blockSize])
	// 创建数组
	orig := make([]byte, len(crypt))
	// 解密
	blockMode.CryptBlocks(orig, crypt)
	// 去补全码
	orig = PKCS7UnPadding(orig)
	return string(orig), nil
}

// PKCS7Padding 补码
func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padText...)
}

// PKCS7UnPadding 去码
func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unPadding := int(origData[length-1])
	return origData[:(length - unPadding)]
}
