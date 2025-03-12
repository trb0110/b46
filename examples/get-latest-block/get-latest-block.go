package main

import (
	"b46/b46/_sys_init"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gagliardetto/solana-go/rpc"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type Response struct {
	JSONRPC string `json:"jsonrpc"`
	Result  struct {
		Context struct {
			APIVersion string `json:"apiVersion"`
			Slot       int    `json:"slot"`
		} `json:"context"`
		Value struct {
			Blockhash            string `json:"blockhash"`
			LastValidBlockHeight int    `json:"lastValidBlockHeight"`
		} `json:"value"`
	} `json:"result"`
	ID int `json:"id"`
}

func main() {
	_ = _sys_init.NewEnviroSetup()

	for attempt := 0; attempt < 5; attempt++ {
		measureExecutionTime("Get Latest 1", GetLatestBlockRpc)
	}
	for attempt := 0; attempt < 5; attempt++ {
		measureExecutionTime("Get Latest 2", GetLatestBlock)
	}
}

func measureExecutionTime(name string, fn func()) {
	start := time.Now()
	fn()
	elapsed := time.Since(start)
	fmt.Printf("%s took %s\n", name, elapsed)
}
func GetLatestBlockRpc() {
	url := _sys_init.Env.RPC
	payload := strings.NewReader("{\"id\":1,\"jsonrpc\":\"2.0\",\"method\":\"getLatestBlockhash\"}")

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()

	var resp Response
	body, _ := io.ReadAll(res.Body)
	if err := json.Unmarshal(body, &resp); err != nil {
		log.Println(err)
	}
	height := resp.Result.Value.LastValidBlockHeight
	fmt.Println("GetLatestBlockRpc lastValidBlockHeight =", height)

}
func GetLatestBlock() {
	rpcClient := rpc.New(_sys_init.Env.RPC)
	recentBlockhash, err := rpcClient.GetLatestBlockhash(context.TODO(), rpc.CommitmentFinalized)
	if err != nil {
		fmt.Errorf("failed to fetch blockhash: %v", err)
	}
	fmt.Println("GetLatestBlock lastValidBlockHeight =", recentBlockhash.Value.LastValidBlockHeight)
}
