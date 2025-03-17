package main

import (
	"b46/b46/_sys_init"
	"b46/b46/models"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/gagliardetto/solana-go/text"
	"log"
	"math"
	"os"
	"strconv"
	"time"
)

const (
	LamportsPerSOL    = models.LamportsPerSOL
	MaxRetries        = models.MaxRetries
	Slippage          = models.Slippage
	TOKEN_DECIMALS    = models.TOKEN_DECIMALS
	tokenMint         = "FqrwAtzZsRSs96azPSkdukBq2V7JXnEuiN52E8zXeswJ"
	bonding           = "5Nj7c92eLJzYXKXHVtbUGzQzJbwbZT5G2QvtgPPoFZdh"
	associatedBonding = "FdXTw2tMThQDTPGSFemjpMJdQfCMqBqpCBbPGdN3tLd7"
)

// BondingCurveState represents the state of a bonding curve
type BondingCurveState struct {
	VirtualTokenReserves uint64
	VirtualSolReserves   uint64
	RealTokenReserves    uint64
	RealSolReserves      uint64
	TokenTotalSupply     uint64
	Complete             bool
}

func main() {

	_ = _sys_init.NewEnviroSetup()
	// Replace these with actual public keys
	mint := solana.MustPublicKeyFromBase58(tokenMint)
	bondingCurve := solana.MustPublicKeyFromBase58(bonding)
	associatedBondingCurve := solana.MustPublicKeyFromBase58(associatedBonding)
	err := executeSell(mint, bondingCurve, associatedBondingCurve)
	if err != nil {
		log.Fatalf("Failed to buy token: %v", err)
	}
}

func executeSell(mint, bondingCurve, associatedBondingCurve solana.PublicKey) error {

	ctx := context.Background()
	// Create a new RPC client:
	rpcClient := rpc.New(_sys_init.Env.RPC)

	//Create a new WS client (used for confirming transactions)
	wsClient, err := ws.Connect(context.Background(), _sys_init.Env.WSS)
	if err != nil {
		panic(err)
	}

	// Decode the private key and create the payer
	payer := solana.MustPrivateKeyFromBase58(_sys_init.Env.PK)

	public := payer.PublicKey()
	log.Println(public)

	// Derive the associated token address
	associatedTokenAddress, _, _ := solana.FindAssociatedTokenAddress(
		payer.PublicKey(),
		mint,
	)
	log.Println(associatedTokenAddress)
	balance, err := getTokenBalance(ctx, rpcClient, associatedTokenAddress)
	if err != nil {
		fmt.Printf("Failed to get token balance: %v\n", err)

	}

	// Convert to a decimal amount (i.e. human-readable token balance)
	tokenBalanceDecimal := float64(balance) / math.Pow10(TOKEN_DECIMALS)
	fmt.Printf("Token balance: %f\n", tokenBalanceDecimal)
	if balance == 0 {
		fmt.Println("No tokens to sell.")

	}
	// Fetch bonding curve state (assumed implemented as getPumpCurveState)
	curveState, err := getPumpCurveState(rpcClient, bondingCurve)
	if err != nil {
		return fmt.Errorf("failed to fetch bonding curve state: %v", err)
	}

	tokenPrice, err := calculatePumpCurvePrice(curveState)
	if err != nil {
		return fmt.Errorf("calculate pump curve: %v", err)
	}

	fmt.Printf("Token Balance: %f\n", tokenBalanceDecimal)
	log.Println("Token price:	", tokenPrice)
	log.Println("Curve State:	", curveState)
	//// Compute Associated Token Address

	// === Calculate minimum SOL output ===
	// Compute the minimum SOL (in lamports) you would accept.
	// First, compute the output in SOL as a float.
	amount := balance
	minSolOutputFloat := tokenBalanceDecimal * tokenPrice
	slippageFactor := 1 - Slippage
	// Multiply by the slippage factor and convert SOL to lamports (and then to an integer).
	minSolOutput := float64((minSolOutputFloat * slippageFactor) * LamportsPerSOL)

	fmt.Printf("Selling %f tokens\n", tokenBalanceDecimal)
	fmt.Printf("Minimum SOL output: %.10f SOL\n", float64(minSolOutput)/LamportsPerSOL)

	time.Sleep(3 * time.Second)
	//Create and send the buy transaction
	errBuy := sellTokenWithRetry(payer, mint, rpcClient, wsClient, associatedTokenAddress, bondingCurve, associatedBondingCurve, amount, minSolOutput)
	if errBuy != nil {
		log.Printf("Failed to create associated token account: %v", errBuy)
	}

	return nil
}

// getTokenBalance retrieves the token balance for the provided associated token account.
// It returns the token balance as an integer, or 0 if no value is found.
func getTokenBalance(ctx context.Context, client *rpc.Client, associatedTokenAccount solana.PublicKey) (int, error) {
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

func sellTokenWithRetry(payer solana.PrivateKey, mint solana.PublicKey, rpcClient *rpc.Client, wsClient *ws.Client, associatedTokenAddress, bondingCurve, associatedBondingCurve solana.PublicKey, tokenAmount int, minSolOutput float64) error {

	var err error
	// Retry up to maxRetries times
	for attempt := 0; attempt < MaxRetries; attempt++ {
		err = sellToken(payer, mint, rpcClient, wsClient, associatedTokenAddress, bondingCurve, associatedBondingCurve, tokenAmount, minSolOutput)
		if err == nil {
			return nil
		}
		// Exponential backoff (2^attempt seconds)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return fmt.Errorf("failed to create associated account after %d retries: %v", MaxRetries, err)
}

func sellToken(payer solana.PrivateKey, mint solana.PublicKey, rpcClient *rpc.Client, wsClient *ws.Client, associatedTokenAddress, bondingCurve, associatedBondingCurve solana.PublicKey, tokenAmount int, minSolOutput float64) error {

	data := make([]byte, 24)

	binary.LittleEndian.PutUint64(data[0:], 12502976635542562355)
	// Write the amount (little-endian uint64) at offset 8.
	binary.LittleEndian.PutUint64(data[8:], uint64(tokenAmount))
	// Write the min_sol_output (little-endian uint64) at offset 16.
	binary.LittleEndian.PutUint64(data[16:], uint64(minSolOutput))

	log.Println(tokenAmount)
	log.Println(minSolOutput)

	pumpProgramPublic := models.PumpProgramPublic
	pumpGlobal := models.PumpGlobal
	pumpFee := models.PumpFee
	systemProgram := models.SystemProgram
	systemTokenProgram := models.SystemTokenProgram
	systemAssociatedTokenAccountProgram := models.SystemAssociatedTokenAccountProgram
	pumpEventAuthority := models.PumpEventAuthority

	sellInstruction := solana.NewInstruction(
		pumpProgramPublic, // PUMP_PROGRAM in the Python code
		solana.AccountMetaSlice{
			{PublicKey: pumpGlobal, IsSigner: false, IsWritable: false},                          // PUMP_GLOBAL
			{PublicKey: pumpFee, IsSigner: false, IsWritable: true},                              // PUMP_FEE
			{PublicKey: mint, IsSigner: false, IsWritable: false},                                // Mint (token mint)
			{PublicKey: bondingCurve, IsSigner: false, IsWritable: true},                         // Bonding curve
			{PublicKey: associatedBondingCurve, IsSigner: false, IsWritable: true},               // Associated bonding curve
			{PublicKey: associatedTokenAddress, IsSigner: false, IsWritable: true},               // Associated token account
			{PublicKey: payer.PublicKey(), IsSigner: true, IsWritable: true},                     // Payer
			{PublicKey: systemProgram, IsSigner: false, IsWritable: false},                       // SYSTEM_PROGRAM
			{PublicKey: systemAssociatedTokenAccountProgram, IsSigner: false, IsWritable: false}, // SYSTEM_TOKEN_PROGRAM
			{PublicKey: systemTokenProgram, IsSigner: false, IsWritable: false},                  // SYSTEM_TOKEN_PROGRAM
			{PublicKey: pumpEventAuthority, IsSigner: false, IsWritable: false},                  // PUMP_EVENT_AUTHORITY
			{PublicKey: pumpProgramPublic, IsSigner: false, IsWritable: false},                   // PUMP_PROGRAM
		},
		data, // Transaction data
	)

	recentBlockhash, err := rpcClient.GetLatestBlockhash(context.TODO(), rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("failed to fetch blockhash: %v", err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{sellInstruction},
		recentBlockhash.Value.Blockhash,
		solana.TransactionPayer(payer.PublicKey()),
	)
	if err != nil {
		return fmt.Errorf("failed to create buy transaction: %v", err)
	}
	tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(payer.PublicKey()) {
			return &payer
		}
		return nil
	})
	spew.Dump(tx)
	// Pretty print the transaction:
	tx.EncodeTree(text.NewTreeEncoder(os.Stdout, "Sell Token"))

	opts := rpc.TransactionOpts{
		SkipPreflight:       true,
		PreflightCommitment: rpc.CommitmentConfirmed,
	}
	t := 3 * time.Second // random default timeout
	timeout := &t
	sig, err := confirm.SendAndConfirmTransactionWithOpts(
		context.TODO(),
		rpcClient,
		wsClient,
		tx,
		opts,
		timeout,
	)
	if err != nil {
		log.Println(err)
	}
	spew.Dump(sig)

	log.Println("Transaction confirmed.")
	return nil
}

func calculateDiscriminator(instructionName string) uint64 {
	// Create a SHA256 hash object
	hash := sha256.New()

	// Update the hash with the instruction name
	hash.Write([]byte(instructionName))

	// Get the first 8 bytes of the hash
	discriminatorBytes := hash.Sum(nil)[:8]

	// Convert the bytes to a 64-bit unsigned integer (little-endian)
	discriminator := binary.LittleEndian.Uint64(discriminatorBytes)

	return discriminator
}

// ParseBondingCurveState parses a byte slice into a BondingCurveState
func ParseBondingCurveState(data []byte) (*BondingCurveState, error) {

	// Skip the first 8 bytes (discriminator)
	offset := 8
	// Parse the fields
	state := &BondingCurveState{
		VirtualTokenReserves: binary.LittleEndian.Uint64(data[offset : offset+8]),
		VirtualSolReserves:   binary.LittleEndian.Uint64(data[offset+8 : offset+16]),
		RealTokenReserves:    binary.LittleEndian.Uint64(data[offset+16 : offset+24]),
		RealSolReserves:      binary.LittleEndian.Uint64(data[offset+24 : offset+32]),
		TokenTotalSupply:     binary.LittleEndian.Uint64(data[offset+32 : offset+40]),
		Complete:             data[offset+40] != 0, // Single byte for the boolean flag
	}

	return state, nil
}

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

// GetPumpCurveState fetches and parses the bonding curve state from an account
func getPumpCurveState(client *rpc.Client, curveAddress solana.PublicKey) (*BondingCurveState, error) {
	// Fetch the account info
	accountInfo, err := client.GetAccountInfo(context.TODO(), curveAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch account info: %v", err)
	}

	accountVal := accountInfo.Value.Data.GetBinary()
	if accountInfo.Value == nil || len(string(accountVal)) == 0 {
		return nil, fmt.Errorf("invalid curve state: no data")
	}

	log.Println(accountVal)
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

// CalculatePumpCurvePrice calculates the price of the token in SOL based on the bonding curve state
func calculatePumpCurvePrice(curveState *BondingCurveState) (float64, error) {
	if curveState.VirtualTokenReserves <= 0 || curveState.VirtualSolReserves <= 0 {
		return 0, fmt.Errorf("invalid reserve state: reserves must be greater than zero")
	}

	tokenPrice := (float64(curveState.VirtualSolReserves) / float64(LamportsPerSOL)) /
		(float64(curveState.VirtualTokenReserves) / math.Pow10(TOKEN_DECIMALS))

	return tokenPrice, nil
}
