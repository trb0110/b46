package sol

import (
	"b46/b46/logging"
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
	"strings"
)

// Field definition for the CreateEvent structure
type Field struct {
	Name string
	Type string
}

func PumpFunListener(conn *websocket.Conn, outputChanel chan<- models.MemeToken) {
	//// Create a context that cancels on SIGINT or SIGTERM.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
		logging.PrintErrorToLog("Failed to marshal subscription message:		", err.Error())
	}
	// Send the subscription message.
	if err := conn.Write(ctx, websocket.MessageText, subBytes); err != nil {
		logging.PrintErrorToLog("Failed tosend subscriptio:		", err.Error())
	}

	log.Println(string(subBytes))
	log.Println("Listening for new token creations from program:  ", models.PumpProgramPublic.String())

	go func() {
		for {
			msgType, msg, err := conn.Read(ctx)
			if err != nil {
				// When context is canceled, the read error is expected.
				logging.PrintErrorToLog("Error reading message:		", err.Error())
				return
			}

			// Process text messages.
			if msgType == websocket.MessageText {

				var response map[string]interface{}

				if err := json.Unmarshal(msg, &response); err != nil {
					logging.PrintErrorToLog("Error unmarshaling JSON:		", err.Error())
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
									logging.PrintErrorToLog("Failed to decode base64:		", err.Error())
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
										meme := ParseTokenInfo(parsedData)
										outputChanel <- meme

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
	close(outputChanel)
	// Optionally, give a moment for cleanup before exit.
	//time.Sleep(1 * time.Second)
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

func ParseTokenInfo(parsedData map[string]string) models.MemeToken {

	meme := models.MemeToken{}
	mint, _ := solana.PublicKeyFromBase58(parsedData["mint"])
	bonding, _ := solana.PublicKeyFromBase58(parsedData["bondingCurve"])
	associatedCurve := FindAssociatedBondingCurve(mint, bonding)

	meme.Name = parsedData["name"]
	meme.Symbol = parsedData["symbol"]
	meme.URI = parsedData["uri"]
	meme.Mint = mint
	meme.BondingCurve = bonding
	meme.AssociatedCurve = associatedCurve
	meme.User = parsedData["user"]
	meme.Info = make([]models.MemeInfo, 0)
	meme.Analysis = make([]models.TokenAnalysis, 0)
	meme.Trading = false
	meme.Sold = false

	return meme
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

func PrintTokenMeme(parsedData models.MemeToken) {

	formattedPrice := fmt.Sprintf("%.20f", parsedData.Info[0].TokenPrice)
	fmt.Println("Name: ", parsedData.Name)
	fmt.Println("Symbol: ", parsedData.Symbol)
	fmt.Println("URI :", parsedData.URI)
	fmt.Println("Mint :", parsedData.Mint)
	fmt.Println("Bonding :", parsedData.BondingCurve)
	fmt.Println("Associated Bonding Curve:", parsedData.AssociatedCurve)
	fmt.Println("Token Price:", formattedPrice)
	fmt.Println("Market Cap:", parsedData.Info[0].MarketCap)
	fmt.Println("Bonding State:", parsedData.Info[0].BondingState)
	fmt.Println("User :", parsedData.User)
	fmt.Println("##########################################################################################")
}
