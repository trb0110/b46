package main

import (
	"b46/b46/_sys_init"
	"b46/b46/logging"
	"b46/b46/models"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"log"
	"math"
)

type AccountInfo struct {
	Balance uint64
	Data    interface{}
	Tokens  map[solana.PublicKey]TokenInfo
}

type TokenInfo struct {
	AssociateAccount solana.PublicKey
	Token            token.Account
}

var ExpectedDiscriminator = func() []byte {
	var value uint64 = 6966180631402821399
	discriminator := make([]byte, 8)
	binary.LittleEndian.PutUint64(discriminator, value)
	return discriminator
}()

func main() {
	_ = _sys_init.NewEnviroSetup()

	rpcClient := rpc.New(_sys_init.Env.RPC)
	payer := solana.MustPrivateKeyFromBase58(_sys_init.Env.PK)

	account := AccountInfo{
		Tokens: make(map[solana.PublicKey]TokenInfo), // 🔹 Initialize the map
	}
	accountInfo, _ := rpcClient.GetAccountInfo(context.TODO(), payer.PublicKey())
	//fmt.Println("Account Info		:", accountInfo.Value)

	accountInfoData := accountInfo.Value.Data.GetBinary()
	dec := bin.NewBinDecoder(accountInfoData)
	err := dec.Decode(&account.Data)
	if err != nil {
		log.Println("fail to decode account info", err)
	}

	account.Balance = accountInfo.Value.Lamports
	fmt.Println("Account 		:", account)

	// 3) Convert lamports -> SOL
	initialSolBalance := float64(account.Balance) / 1e9
	totalSOL := float64(account.Balance) / 1e9
	fmt.Printf("Initial SOL Balance: %.20f SOL\n", initialSolBalance)

	out, err := rpcClient.GetTokenAccountsByOwner(
		context.TODO(),
		payer.PublicKey(),
		&rpc.GetTokenAccountsConfig{
			ProgramId: &solana.TokenProgramID,
		},
		&rpc.GetTokenAccountsOpts{
			Encoding: solana.EncodingBase64Zstd,
		},
	)

	if err != nil {
		fmt.Println("failed to get tokens: %v", err)
	}

	for _, rawAccount := range out.Value {
		var tokenAccount TokenInfo
		var tokAcc token.Account

		log.Println(rawAccount)
		data := rawAccount.Account.Data.GetBinary()

		dec := bin.NewBinDecoder(data)
		err := dec.Decode(&tokAcc)
		if err != nil {
			panic(err)
		}
		//log.Println(tokAcc)

		tokenAccount.AssociateAccount = rawAccount.Pubkey
		tokenAccount.Token = tokAcc
		account.Tokens[tokenAccount.Token.Mint] = tokenAccount
	}

	fmt.Println(account)

	// 6) Now convert each token to SOL
	//    We must fetch:
	//      a) the token mint’s decimals
	//      b) the price in SOL (or fetch price in USD and convert USD->SOL).
	for _, tk := range account.Tokens {
		mint := tk.Token.Mint
		// Derive the bonding curve address (and get the bump seed).
		bondingCurveAddress, bump, _ := GetBondingCurveAddress(mint, models.PumpProgramPublic)

		// Calculate the associated bonding curve address.
		associatedBondingCurve := FindAssociatedBondingCurve(mint, bondingCurveAddress)

		fmt.Println("------------------ACCOUNT INFO----------------------")
		fmt.Printf("Token Mint:               %s \n", mint.String())
		fmt.Printf("Bonding Curve:            %s \n", bondingCurveAddress.String())
		fmt.Printf("Associated Bonding Curve: %s \n", associatedBondingCurve.String())
		fmt.Printf("Bonding Curve Bump:       %d \n", bump)
		fmt.Printf("Amount Held:       %d \n", tk.Token.Amount)

		curveState, err := GetPumpCurveState(rpcClient, bondingCurveAddress)
		if err != nil {
			logging.PrintErrorToLog("failed to fetch bonding curve state:		", err.Error())
		}
		fmt.Println("Curve State:		", curveState)
		tokenPrice, err := CalculatePumpCurvePrice(curveState)
		if err != nil {
			logging.PrintErrorToLog("failed to calculate pump curve price:		", err.Error())
		}

		fmt.Println("--------------------------------------------------")
		totalSOL += tokenPrice
		//fmt.Printf("Token mint %v => %f tokens => %f SOL\n", mint, quantity, tokenValueInSOL)
	}

	fmt.Printf("Initial SOL Balance: %.20f SOL\n", initialSolBalance)
	fmt.Printf("Final SOL Balance: %.20f SOL\n", totalSOL)
	//log.Println(account)

}

// getMintDecimals queries the mint account to get the `decimals` field.
// In the token standard, that's a single byte in the mint data.
func getMintDecimals(client *rpc.Client, mint solana.PublicKey) (uint8, error) {
	accountInfo, err := client.GetAccountInfo(context.TODO(), mint)
	if err != nil {
		return 0, err
	}
	if accountInfo.Value == nil {
		return 0, fmt.Errorf("mint account not found: %v", mint)
	}

	// Mint layout is a TokenMint struct with certain offsets.
	// Using gagliardetto/solana-go token.Mint struct, for example:
	var mintAccount token.Mint

	data := accountInfo.Value.Data.GetBinary()
	dec := bin.NewBinDecoder(data)
	if err := dec.Decode(&mintAccount); err != nil {
		return 0, fmt.Errorf("failed to decode mint: %w", err)
	}

	return mintAccount.Decimals, nil
}

// getTokenPriceInSOL is a placeholder for how you'd get the token price in SOL.
// Realistically, you'd query an oracle (Switchboard / Pyth), a DEX aggregator, or
// some third-party API that provides price data in SOL or in USD then convert to SOL.
func getTokenPriceInSOL(mint solana.PublicKey) (float64, error) {
	// Example: For demonstration, every token is worth 0.0001 SOL
	// Replace with real logic: fetch from a price feed or aggregator
	return 0.0001, nil
}

// CalculatePumpCurvePrice calculates the price of the token in SOL based on the bonding curve state
func CalculatePumpCurvePrice(curveState *models.BondingCurveState) (float64, error) {
	if curveState.VirtualTokenReserves <= 0 || curveState.VirtualSolReserves <= 0 {
		return 0, fmt.Errorf("invalid reserve state: reserves must be greater than zero")
	}

	tokenPrice := (float64(curveState.VirtualSolReserves) / float64(models.LamportsPerSOL)) /
		(float64(curveState.VirtualTokenReserves) / math.Pow10(models.TOKEN_DECIMALS))

	return tokenPrice, nil
}

// GetPumpCurveState fetches and parses the bonding curve state from an account
func GetPumpCurveState(client *rpc.Client, curveAddress solana.PublicKey) (*models.BondingCurveState, error) {
	// Fetch the account info
	accountInfo, err := client.GetAccountInfo(context.Background(), curveAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch account info: %v", err)
	}

	accountVal := accountInfo.Value.Data.GetBinary()
	if accountInfo.Value == nil || len(string(accountVal)) == 0 {
		return nil, fmt.Errorf("invalid curve state: no data")
	}

	// Check the discriminator
	if !bytes.Equal(accountVal[:8], ExpectedDiscriminator) {
		return nil, fmt.Errorf("invalid curve state discriminator")
	}

	// Parse the bonding curve state
	state, err := ParseBondingCurveState(accountVal)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bonding curve state: %v", err)
	}

	return state, nil
}

// ParseBondingCurveState parses a byte slice into a BondingCurveState
func ParseBondingCurveState(data []byte) (*models.BondingCurveState, error) {

	// Skip the first 8 bytes (discriminator)
	offset := 8
	// Parse the fields
	state := &models.BondingCurveState{
		VirtualTokenReserves: binary.LittleEndian.Uint64(data[offset : offset+8]),
		VirtualSolReserves:   binary.LittleEndian.Uint64(data[offset+8 : offset+16]),
		RealTokenReserves:    binary.LittleEndian.Uint64(data[offset+16 : offset+24]),
		RealSolReserves:      binary.LittleEndian.Uint64(data[offset+24 : offset+32]),
		TokenTotalSupply:     binary.LittleEndian.Uint64(data[offset+32 : offset+40]),
		Complete:             data[offset+40] != 0, // Single byte for the boolean flag
	}

	return state, nil
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

// GetBondingCurveAddress derives the bonding curve address for a given mint.
// It uses two seeds: the literal "bonding-curve" and the mint’s bytes.
func GetBondingCurveAddress(mint solana.PublicKey, programID solana.PublicKey) (solana.PublicKey, uint8, error) {
	seeds := [][]byte{
		[]byte("bonding-curve"),
		mint.Bytes(),
	}
	// solana.FindProgramAddress returns (pubkey, bump).
	return solana.FindProgramAddress(seeds, programID)
}
