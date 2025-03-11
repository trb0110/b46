package main

import (
	"b46/b46/_sys_init"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/coder/websocket"
	"log"
	"time"
)

func main() {
	_ = _sys_init.NewEnviroSetup()
	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, _sys_init.Env.WSS, nil)
	if err != nil {
		log.Printf("Connection error: %v\nReconnecting in 5 seconds...", err)
		time.Sleep(5 * time.Second)
		return
	}
	//log.Println(conn)
	defer conn.Close(websocket.StatusInternalError, "Connection closed")

	accountPublicKey := "2AY6h9hTPq2tHEwrHVuZVsfhMQvRUTFNb7BsRzxRpcT6"
	subscriptionMessage := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "accountSubscribe",
		"params": []interface{}{
			accountPublicKey,
			map[string]interface{}{
				"commitment": "confirmed",
				"encoding":   "jsonParsed",
			},
		},
	}

	//log.Println(subscriptionMessage)
	subscriptionBytes, _ := json.Marshal(subscriptionMessage)
	if err := conn.Write(ctx, websocket.MessageText, subscriptionBytes); err != nil {
		log.Printf("Failed to send subscription: %v\n", err)
		return
	}

	log.Println(string(subscriptionBytes))
	log.Println("Listening for account activity...")

	_, message, err := conn.Read(ctx)
	if err != nil {
		log.Printf("Error reading message: %v\n", err)
		return
	}

	log.Println("Subscription response: ", string(message))

	for {
		_, message, err := conn.Read(ctx)
		if err != nil {
			log.Printf("Error reading message: %v\n", err)
			return
		}
		var response map[string]interface{}

		// Unmarshal the JSON response
		err2 := json.Unmarshal(message, &response)
		if err2 != nil {
			log.Fatalf("Error unmarshaling JSON: %v\n", err2)
		}
		log.Println(response)

		params, _ := response["params"].(map[string]interface{})
		result, _ := params["result"].(map[string]interface{})
		value, _ := result["value"].(map[string]interface{})
		data, _ := value["data"].([]interface{})

		log.Println(data[0])
		log.Println(data[1])

		decodedData, err := base64.StdEncoding.DecodeString(data[1].(string))
		if err != nil {
			log.Printf("Failed to decode log: %v\n", err)
			continue
		}

		log.Println(decodedData)

	}
}
