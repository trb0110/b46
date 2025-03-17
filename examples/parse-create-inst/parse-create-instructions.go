package main

import (
	"b46/b46/models"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
	"log"
)

// Field definition for the CreateEvent structure
type Field struct {
	Name string
	Type string
}

func main() {

	encodedData := "G3KpTd7rY3YGAAAATWNQdW1wBgAAAE1jUHVtcEMAAABodHRwczovL2lwZnMuaW8vaXBmcy9RbVJzaDN0TFU1M2tUQ040VnVIakpybmVybzZDUnpTNEpQNGJRS3NzQlBBZjVB89K0UySuzclikD+XPtC1H8QpoU205cIsuq8pCS4vAI9G2trpaA4mS1zeVaKJ5g3pzXTYXqNFaQNqB70aWqB7aYEfHNsLKMDhwRdwgUUpXFpKdkrgnZNvL59MB/1Islgu"

	log.Println(encodedData)
	decodedData, err := base64.StdEncoding.DecodeString(encodedData)
	if err != nil {
		log.Printf("Failed to decode log: %v\n", err)
	}

	//log.Println(string(decodedData))
	parsedData := ParseCreateInstruction(decodedData)
	if parsedData != nil {
		PrintTokenInfo(parsedData)
	}
}

// ParseCreateInstruction parses the "create" instruction data
func ParseCreateInstruction(data []byte) map[string]string {
	if len(data) < 8 {
		return nil
	}
	offset := 8
	parsedData := make(map[string]string)

	fields := []Field{
		{"name", "string"},
		{"symbol", "string"},
		{"uri", "string"},
		{"mint", "publicKey"},
		{"bondingCurve", "publicKey"},
		{"user", "publicKey"},
	}

	for _, field := range fields {
		if field.Type == "string" {
			if offset+4 > len(data) {
				return nil
			}
			length := binary.LittleEndian.Uint32(data[offset : offset+4])
			offset += 4
			if offset+int(length) > len(data) {
				return nil
			}
			value := string(data[offset : offset+int(length)])
			offset += int(length)
			parsedData[field.Name] = value
		} else if field.Type == "publicKey" {
			if offset+32 > len(data) {
				return nil
			}
			value := base58.Encode(data[offset : offset+32])
			offset += 32
			parsedData[field.Name] = value
		}
	}

	return parsedData
}

func PrintTokenInfo(parsedData map[string]string) {

	mint, _ := solana.PublicKeyFromBase58(parsedData["mint"])
	bonding, _ := solana.PublicKeyFromBase58(parsedData["bondingCurve"])
	associatedCurve := FindAssociatedBondingCurve(mint, bonding)

	log.Println("New Token Created:")
	fmt.Println("Name: ", parsedData["name"])
	fmt.Println("Symbol: ", parsedData["symbol"])
	fmt.Println("URI :", parsedData["uri"])
	fmt.Println("Mint :", parsedData["mint"])
	fmt.Println("Bonding :", parsedData["bondingCurve"])
	fmt.Println("Associated Bonding Curve:", associatedCurve)
	fmt.Println("User :", parsedData["user"])
	fmt.Println("##########################################################################################")
}

// FindAssociatedBondingCurve derives the associated bonding curve address (using the ATA derivation)
// from the bonding curve and the mint address. The seeds used here are:
// bondingCurve, token program, and mint.
func FindAssociatedBondingCurve(mint solana.PublicKey, bondingCurve solana.PublicKey) solana.PublicKey {
	seeds := [][]byte{
		bondingCurve.Bytes(),
		models.SystemTokenProgram.Bytes(),
		mint.Bytes(),
	}
	derivedAddress, _, _ := solana.FindProgramAddress(seeds, models.SystemAssociatedTokenAccountProgram)
	return derivedAddress
}
