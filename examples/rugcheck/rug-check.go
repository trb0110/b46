package main

import (
	"b46/b46/_sys_init"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// TokenMint holds the parsed fields from a token mint account.
type TokenMint struct {
	Decimals        int         `json:"decimals"`
	FreezeAuthority interface{} `json:"freezeAuthority"`
	IsInitialized   bool        `json:"isInitialized"`
	MintAuthority   interface{} `json:"mintAuthority"`
	Supply          string      `json:"supply"`
}

type AccountInfo struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Context struct {
			ApiVersion string `json:"apiVersion"`
			Slot       int    `json:"slot"`
		} `json:"context"`
		Value struct {
			Data struct {
				Parsed struct {
					Info TokenMint `json:"info"`
					Type string    `json:"type"`
				} `json:"parsed"`
				Program string `json:"program"`
				Space   int    `json:"space"`
			} `json:"data"`
			Executable bool    `json:"executable"`
			Lamports   int     `json:"lamports"`
			Owner      string  `json:"owner"`
			RentEpoch  float64 `json:"rentEpoch"`
			Space      int     `json:"space"`
		} `json:"value"`
	} `json:"result"`
	Id int `json:"id"`
}

// computeRiskScore computes a risk score out of 10 using some simple heuristics.
// These are just example rules:
//   - If a mint authority is present, add 2 points (the team can mint more tokens).
//   - If a freeze authority is present, add 2 points (the team can freeze token transfers).
//   - If the normalized token supply (supply / 10^decimals) is huge (> 1e6), add 3 points.
//   - If decimals is unusually high (> 9), add 1 point.
//
// The total is capped at 10.
func computeRiskScore(tm TokenMint) float64 {
	score := 10.0

	// Having a mint authority means more tokens can be minted.
	if tm.MintAuthority != nil && tm.MintAuthority != "" {
		score -= 2
	}
	// A freeze authority gives the team power to freeze accounts.
	if tm.FreezeAuthority != nil && tm.FreezeAuthority != "" {
		score -= 2
	}
	supp, _ := strconv.ParseFloat(tm.Supply, 64)
	// Normalize supply: convert raw supply to human-friendly amount.
	normalizedSupply := supp / math.Pow(10, float64(tm.Decimals))
	if normalizedSupply > 1e9 { // arbitrarily, if more than 1 million tokens exist.
		score -= 3
	}
	// If decimals are unusually high, add a small risk factor.
	if tm.Decimals > 9 {
		score -= 2
	}

	return score
}

func main() {

	_ = _sys_init.NewEnviroSetup()
	//log.Println(_sys_init.Env)
	// Parse the token mint address from the command-line.

	payload := strings.NewReader("{\"id\":1,\"jsonrpc\":\"2.0\",\"method\":\"getAccountInfo\",\"params\":[\"CyCsGm38rqkJnWrJat6LJV6ZRRVPMLXu3D6PhWQc5wK7\",{\"encoding\":\"jsonParsed\",\"commitment\":\"finalized\"}]}")

	req, _ := http.NewRequest("POST", _sys_init.Env.RPC, payload)

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)

	if err != nil {
		log.Fatalf("Failed to get account info: %v", err)
	}

	if len(body) < 1 {
		log.Fatalf("Invalid account data format")

	}
	var accountInfo AccountInfo

	unmarshalError := json.Unmarshal(body, &accountInfo)
	if unmarshalError != nil {
		log.Println("Failed to unmarshal account info: %v", unmarshalError)
	}

	// Parse the token mint data.
	tokenMint := accountInfo.Result.Value.Data.Parsed.Info
	// Print the token mint details.
	fmt.Println("Token Mint Details:")
	if tokenMint.MintAuthority != nil {
		fmt.Printf("  Mint Authority:   %s\n", tokenMint.MintAuthority)
	} else {
		fmt.Println("  Mint Authority:   None")
	}
	fmt.Printf("  Supply (raw):     %s\n", tokenMint.Supply)

	supp, _ := strconv.ParseFloat(tokenMint.Supply, 64)
	normalizedSupply := supp / math.Pow(10, float64(tokenMint.Decimals))
	fmt.Printf("  Supply (normalized): %.2f\n", normalizedSupply)
	fmt.Printf("  Decimals:         %d\n", tokenMint.Decimals)
	fmt.Printf("  Is Initialized:   %v\n", tokenMint.IsInitialized)
	if tokenMint.FreezeAuthority != nil {
		fmt.Printf("  Freeze Authority: %s\n", tokenMint.FreezeAuthority)
	} else {
		fmt.Println("  Freeze Authority: None")
	}

	// Compute and display a risk score out of 10.
	riskScore := computeRiskScore(tokenMint)
	fmt.Printf("\nRug Check Risk Score: %.1f/10\n", riskScore)
}
