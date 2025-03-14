package main

import (
	"b46/b46/_sys_init"
	"b46/b46/logging"
	"b46/b46/models"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"math"
)

const (
	mintAddress = "8VfUQdY8S5DFnCPXUbP8hTxdEM1wWYbYoU9p1aoPpump"
)

var ExpectedDiscriminator = func() []byte {
	var value uint64 = 6966180631402821399
	discriminator := make([]byte, 8)
	binary.LittleEndian.PutUint64(discriminator, value)
	return discriminator
}()

var Discriminator = func() []byte {
	var value uint64 = 16927863322537952870
	discriminator := make([]byte, 8)
	binary.LittleEndian.PutUint64(discriminator, value)
	return discriminator
}()

func main() {
	_ = _sys_init.NewEnviroSetup()

	rpcClient := rpc.New(_sys_init.Env.RPC)
	// Parse the mint address from Base58.
	mint, err := solana.PublicKeyFromBase58(mintAddress)
	if err != nil {
		fmt.Printf("Error: Invalid address format - %s\n", err)
		return
	}

	// Derive the bonding curve address (and get the bump seed).
	bondingCurveAddress, bump, _ := GetBondingCurveAddress(mint, models.PumpProgramPublic)

	// Calculate the associated bonding curve address.
	associatedBondingCurve := FindAssociatedBondingCurve(mint, bondingCurveAddress)

	fmt.Println("\nResults:")
	fmt.Println("--------------------------------------------------")
	fmt.Printf("Token Mint:               %s\n", mint.String())
	fmt.Printf("Bonding Curve:            %s\n", bondingCurveAddress.String())
	fmt.Printf("Associated Bonding Curve: %s\n", associatedBondingCurve.String())
	fmt.Printf("Bonding Curve Bump:       %d\n", bump)
	fmt.Println("--------------------------------------------------")

	curveState, err := GetPumpCurveState(rpcClient, bondingCurveAddress)
	if err != nil {
		logging.PrintErrorToLog("failed to fetch bonding curve state:		", err.Error())
	}
	fmt.Println("\nCurve State:		", curveState)
	tokenPrice, err := CalculatePumpCurvePrice(curveState)
	if err != nil {
		logging.PrintErrorToLog("failed to calculate pump curve price:		", err.Error())
	}
	//marketCap := GetTokenMarketCap(curveState, tokenPrice)

	fmt.Println("\nToken Price:		", tokenPrice)

	//fmt.Println("\nMarket Cap:		", marketCap)
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
