package helpers

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
)

const esKeyFilePath = "./es-key.key"

func GeneratesLlavesES() error {
	if _, errPath := os.Stat(esKeyFilePath); errPath == nil {
		return nil
	}

	keyBytes := make([]byte, 32)
	if _, errRand := rand.Read(keyBytes); errRand != nil {
		return errRand
	}

	fileKey, errCreate := os.Create(esKeyFilePath)
	if errCreate != nil {
		log.Fatal("Error al crear el archivo que contendra la llave")
	}
	defer fileKey.Close()

	if _, errWrite := fileKey.WriteString(hex.EncodeToString(keyBytes)); errWrite != nil {
		return errWrite
	}

	return nil
}

func LoadLlavesES() ([]byte, error) {
	fileKey, errRead := os.ReadFile(esKeyFilePath)
	if errRead != nil {
		return nil, errRead
	}

	keyBytes, errDecode := hex.DecodeString(string(fileKey))
	if errDecode != nil {
		return nil, errDecode
	}

	return keyBytes, nil
}
