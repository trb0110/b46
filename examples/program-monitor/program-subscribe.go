package main

import (
	"b46/b46/_sys_init"
	"b46/b46/models"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Field definition for the CreateEvent structure
type Field struct {
	Name string
	Type string
}

func main() {
	_ = _sys_init.NewEnviroSetup()
	// Create a context that cancels on SIGINT or SIGTERM.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for interrupt signals to gracefully shut down.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Printf("Received signal: %v. Shutting down...", sig)
		cancel()
	}()

	conn, _, err := websocket.Dial(ctx, _sys_init.Env.WSS, nil)
	if err != nil {
		log.Fatalf("Failed to connect to websocket: %v", err)
	}
	//log.Println(conn)
	defer conn.Close(websocket.StatusInternalError, "Connection closed")

	subscriptionMessage := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "logsSubscribe",
		"params": []interface{}{
			map[string]interface{}{
				"mentions": []string{models.PumpProgramPublic.String()},
			},
			map[string]interface{}{
				"commitment": "processed",
			},
		},
	}

	// Marshal the subscription message to JSON.
	subBytes, err := json.Marshal(subscriptionMessage)
	if err != nil {
		log.Fatalf("Failed to marshal subscription message: %v", err)
	}
	// Send the subscription message.
	if err := conn.Write(ctx, websocket.MessageText, subBytes); err != nil {
		log.Fatalf("Failed to send subscription: %v", err)
	}

	log.Println(string(subBytes))
	log.Println("Listening for new token creations from program:  ", models.PumpProgramPublic.String())

	go func() {
		for {
			msgType, msg, err := conn.Read(ctx)
			if err != nil {
				// When context is canceled, the read error is expected.
				log.Printf("Error reading message: %v", err)
				return
			}

			// Process text messages.
			if msgType == websocket.MessageText {

				var response map[string]interface{}

				if err := json.Unmarshal(msg, &response); err != nil {
					log.Printf("Error unmarshaling JSON: %v", err)
				}

				// Check for a logs notification.
				if method, ok := response["method"].(string); ok && method == "logsNotification" {
					// Drill down into the JSON structure.
					params, ok := response["params"].(map[string]interface{})
					if !ok {
						continue
					}
					result, ok := params["result"].(map[string]interface{})
					if !ok {
						continue
					}
					value, ok := result["value"].(map[string]interface{})
					if !ok {
						continue
					}

					logsIface, ok := value["logs"].([]interface{})
					if !ok {
						continue
					}

					createFound := false
					for _, logItem := range logsIface {
						logStr, ok := logItem.(string)
						if !ok {
							continue
						}
						if strings.Contains(logStr, "Program log: Instruction: Create") {
							createFound = true
							break
						}
					}

					if createFound {
						// Process each log that contains "Program data:".
						for _, logItem := range logsIface {
							logStr, ok := logItem.(string)
							if !ok {
								continue
							}
							if strings.Contains(logStr, "Program data:") {
								parts := strings.SplitN(logStr, ": ", 2)
								if len(parts) < 2 {
									continue
								}
								encodedData := parts[1]
								// Decode the Base64 data.
								decodedData, err := base64.StdEncoding.DecodeString(encodedData)
								if err != nil {
									log.Printf("Failed to decode base64: %v", err)
									continue
								}

								parsedData := ParseCreateInstruction(decodedData)
								if parsedData != nil {
									if name, ok := parsedData["name"]; ok && name != "" {
										// Print signature if available.

										log.Println("New Token Created:")

										if sig, ok := value["signature"].(string); ok {
											fmt.Println("Signature:", sig)
										}
										PrintTokenInfo(parsedData)

									}
								}
							}
						}
					}
				} else {
					log.Printf("Received message: %s", msg)
				}

			} else {
				log.Printf("Received non-text message of type %d", msgType)
			}
		}
	}()

	// Keep the main function running until the context is canceled.
	<-ctx.Done()

	// Optionally, give a moment for cleanup before exit.
	time.Sleep(1 * time.Second)
	log.Println("Exiting program.")

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

func PrintTokenInfo(parsedData map[string]string) {

	mint, _ := solana.PublicKeyFromBase58(parsedData["mint"])
	bonding, _ := solana.PublicKeyFromBase58(parsedData["bondingCurve"])
	associatedCurve := FindAssociatedBondingCurve(mint, bonding)

	fmt.Println("Name: ", parsedData["name"])
	fmt.Println("Symbol: ", parsedData["symbol"])
	fmt.Println("URI :", parsedData["uri"])
	fmt.Println("Mint :", parsedData["mint"])
	fmt.Println("Bonding :", parsedData["bondingCurve"])
	fmt.Println("Associated Bonding Curve:", associatedCurve)
	fmt.Println("User :", parsedData["user"])
	fmt.Println("##########################################################################################")
}
