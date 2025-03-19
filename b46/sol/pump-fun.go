package sol

import (
	"b46/b46/models"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"math"
	"math/big"
	"strconv"
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

// CalculatePumpCurvePrice calculates the price of the token in SOL based on the bonding curve state
func CalculatePumpCurvePrice(curveState *models.BondingCurveState) (float64, error) {
	if curveState.VirtualTokenReserves <= 0 || curveState.VirtualSolReserves <= 0 {
		return 0, fmt.Errorf("invalid reserve state: reserves must be greater than zero")
	}

	tokenPrice := (float64(curveState.VirtualSolReserves) / float64(models.LamportsPerSOL)) /
		(float64(curveState.VirtualTokenReserves) / math.Pow10(models.TOKEN_DECIMALS))

	return tokenPrice, nil
}

// GetTokenMarketCap calculates the price of the token in SOL based on the bonding curve state
func GetTokenMarketCap(curveState *models.BondingCurveState, tokenPrice float64) float64 {

	// Convert to float
	totalSupply := float64(curveState.TokenTotalSupply / 1000000)

	// Market Cap = Price * Supply
	marketCap := new(big.Float).Mul(big.NewFloat(tokenPrice), big.NewFloat(totalSupply))

	// Convert to float64
	marketCapFloat, _ := marketCap.Float64()

	return marketCapFloat
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

// getTokenBalance retrieves the token balance for the provided associated token account.
// It returns the token balance as an integer, or 0 if no value is found.
func GetTokenBalance(ctx context.Context, client *rpc.Client, associatedTokenAccount solana.PublicKey) (int, error) {
	// The GetTokenAccountBalance method expects the public key as a string.
	resp, err := client.GetTokenAccountBalance(ctx, associatedTokenAccount, rpc.CommitmentFinalized)
	if err != nil {
		return 0, fmt.Errorf("error getting token account balance: %w", err)
	}

	if resp.Value != nil {
		// Convert the amount (a string) to an integer.
		balance, err := strconv.Atoi(resp.Value.Amount)
		if err != nil {
			return 0, fmt.Errorf("error converting token balance to int: %w", err)
		}
		return balance, nil
	}

	// If the response value is nil, return 0.
	return 0, nil
}
