package main

import (
	"b46/b46/_sys_init"
	"b46/b46/models"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/rpc"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/gagliardetto/solana-go/text"
	"log"
	"math"
	"os"
	"strings"
	"time"
)

const (
	LamportsPerSOL     = models.LamportsPerSOL
	MaxRetries         = models.MaxRetries
	Slippage           = models.Slippage
	TOKEN_DECIMALS     = models.TOKEN_DECIMALS
	tokenMint          = "6mDT8DLcYwSrrzZHf1EXM7mEr6QLmkEHK1uKM4xCpump"
	bonding            = "J32cjuosoZfEvJwtaMtCXWEJrt3Eq3jbed3PjrGNJXZS"
	associatedBonding  = "H17YSm69yNXJuWuBrdqT6BMZt4g2wu1wVCbKbiM2Cwov"
	PriorityFeeLamport = 10000
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
	mint := solana.MustPublicKeyFromBase58(tokenMint)
	bondingCurve := solana.MustPublicKeyFromBase58(bonding)
	associatedBondingCurve := solana.MustPublicKeyFromBase58(associatedBonding)
	amount := 0.00001
	err := executeBuy(mint, bondingCurve, associatedBondingCurve, amount)
	if err != nil {
		log.Fatalf("Failed to buy token: %v", err)
	}
}

// executeBuy is the top‐level function that performs the buy pipeline.
func executeBuy(mint, bondingCurve, associatedBondingCurve solana.PublicKey, amount float64) error {
	// Create a new RPC client:
	rpcClient := rpc.New(_sys_init.Env.RPC)

	// Create a new WS client (used for confirming transactions)
	wsClient, err := ws.Connect(context.Background(), _sys_init.Env.WSS)
	if err != nil {
		panic(err)
	}

	defer wsClient.Close()

	// Decode the private key and create the payer
	payer := solana.MustPrivateKeyFromBase58(_sys_init.Env.PK)
	amountLamports := int(amount * LamportsPerSOL)

	log.Println("Payer public key:", payer.PublicKey())

	// (1) Pre-fetch bonding curve state concurrently.
	curveState, err := getPumpCurveState(rpcClient, bondingCurve)
	if err != nil {
		return fmt.Errorf("failed to fetch bonding curve state: %v", err)
	}

	// (2) Calculate token price and amount.
	tokenPrice, err := calculatePumpCurvePrice(curveState)
	if err != nil {
		return fmt.Errorf("failed to calculate pump curve price: %v", err)
	}
	tokenAmount := amount / tokenPrice
	maxAmountLamports := uint64(float64(amountLamports) * (1 + Slippage))

	log.Println("Token price:", tokenPrice)
	log.Println("Token Amount:", tokenAmount)
	log.Println("Max lamports:", maxAmountLamports)
	log.Println("Curve State:", curveState)

	// (3) Derive associated token account.
	associatedTokenAddress, _, err := solana.FindAssociatedTokenAddress(
		payer.PublicKey(),
		mint,
	)
	if err != nil {
		return fmt.Errorf("failed to derive associated token address: %v", err)
	}
	log.Println("Associated token account:", associatedTokenAddress)

	// (4) Check if the associated token account exists; if not, create it.
	_, errAssociated := rpcClient.GetAccountInfo(context.TODO(), associatedTokenAddress)
	if errAssociated != nil {
		if strings.Contains(errAssociated.Error(), "not found") {
			errCreateAssociated := createAssociatedAccountWithRetry(payer, mint, rpcClient, wsClient, associatedTokenAddress)
			if errCreateAssociated != nil {
				log.Printf("Failed to create associated token account: %v", errCreateAssociated)
				return errCreateAssociated
			}
		} else {
			log.Printf("Unexpected error checking associated token account: %v", errAssociated)
		}
	}

	// (5) Wait briefly for the associated account creation to propagate.
	time.Sleep(3 * time.Second)

	// (6) Create and send the buy transaction (with retries).
	errBuy := buyTokenWithRetry(payer, mint, rpcClient, wsClient, associatedTokenAddress, bondingCurve, associatedBondingCurve, tokenAmount, maxAmountLamports)
	if errBuy != nil {
		log.Printf("Buy transaction failed: %v", errBuy)
		return errBuy
	}

	return nil
}

// buyTokenWithRetry calls buyToken with retry logic.
func buyTokenWithRetry(payer solana.PrivateKey, mint solana.PublicKey, rpcClient *rpc.Client, wsClient *ws.Client, associatedTokenAddress, bondingCurve, associatedBondingCurve solana.PublicKey, tokenAmount float64, maxAmountLamports uint64) error {
	var err error
	for attempt := 0; attempt < MaxRetries; attempt++ {
		err = buyToken(payer, mint, rpcClient, wsClient, associatedTokenAddress, bondingCurve, associatedBondingCurve, tokenAmount, maxAmountLamports)
		if err == nil {
			return nil
		}
		log.Printf("Buy attempt %d failed: %v. Retrying...", attempt+1, err)
		time.Sleep(time.Duration(1<<attempt) * time.Second) // exponential backoff
	}
	return fmt.Errorf("failed to execute buy after %d retries: %v", MaxRetries, err)
}

// buyToken builds, simulates, and sends the buy transaction.
// It uses the ComputeBudget instruction for priority fees and performs a simulation pre-check.
func buyToken(payer solana.PrivateKey, mint solana.PublicKey, rpcClient *rpc.Client, wsClient *ws.Client, associatedTokenAddress, bondingCurve, associatedBondingCurve solana.PublicKey, tokenAmount float64, maxAmountLamports uint64) error {
	ctx := context.TODO()

	// Prepare the instruction data.
	data := append(Discriminator, make([]byte, 16)...)
	// tokenAmount is converted to token units (e.g. if token has 6 decimals, multiply by 1e6)
	binary.LittleEndian.PutUint64(data[8:], uint64(tokenAmount*1e6))
	binary.LittleEndian.PutUint64(data[16:], maxAmountLamports)

	// Build the buy instruction.
	buyInstruction := solana.NewInstruction(
		models.PumpProgramPublic, // PUMP_PROGRAM (from your models)
		solana.AccountMetaSlice{
			{PublicKey: models.PumpGlobal, IsSigner: false, IsWritable: false},         // PUMP_GLOBAL
			{PublicKey: models.PumpFee, IsSigner: false, IsWritable: true},             // PUMP_FEE
			{PublicKey: mint, IsSigner: false, IsWritable: false},                      // Mint
			{PublicKey: bondingCurve, IsSigner: false, IsWritable: true},               // Bonding curve
			{PublicKey: associatedBondingCurve, IsSigner: false, IsWritable: true},     // Associated bonding curve
			{PublicKey: associatedTokenAddress, IsSigner: false, IsWritable: true},     // Associated token account
			{PublicKey: payer.PublicKey(), IsSigner: true, IsWritable: true},           // Payer
			{PublicKey: models.SystemProgram, IsSigner: false, IsWritable: false},      // SYSTEM_PROGRAM
			{PublicKey: models.SystemTokenProgram, IsSigner: false, IsWritable: false}, // SYSTEM_TOKEN_PROGRAM
			{PublicKey: models.SystemRent, IsSigner: false, IsWritable: false},         // SYSTEM_RENT
			{PublicKey: models.PumpEventAuthority, IsSigner: false, IsWritable: false}, // PUMP_EVENT_AUTHORITY
			{PublicKey: models.PumpProgramPublic, IsSigner: false, IsWritable: false},  // PUMP_PROGRAM again as required
		},
		data,
	)

	// Get getRecentPrioritizationFees
	// Set Compute Unit Limit (300 units)
	//modifyComputeUnits := computebudget.NewSetComputeUnitPriceInstruction(300)

	// Add a compute budget instruction to set a higher compute unit price (priority fee).
	// This helps ensure your transaction is processed faster in a congested network.
	priorityIx, errPriority := computebudget.NewSetComputeUnitPriceInstruction(PriorityFeeLamport).ValidateAndBuild()
	if errPriority != nil {
		return fmt.Errorf("failed to set priority: %v", errPriority)
	}

	// Pre-fetch a recent blockhash (to avoid stale blockhash issues).
	blockhashResp, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("failed to fetch blockhash: %v", err)
	}
	blockhash := blockhashResp.Value.Blockhash

	// Build the transaction with the compute budget instruction added at the beginning.
	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			priorityIx,     // Priority fee instruction
			buyInstruction, // Main buy instruction
		},
		blockhash,
		solana.TransactionPayer(payer.PublicKey()),
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %v", err)
	}

	// Sign the transaction.
	tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(payer.PublicKey()) {
			return &payer
		}
		return nil
	})

	// Simulate the transaction before sending.
	//simOpts := rpc.SimulateTransactionOpts{
	//	SigVerify:              true,
	//	Commitment:             rpc.CommitmentProcessed,
	//	ReplaceRecentBlockhash: true,
	//}

	simResult, errSim := rpcClient.SimulateTransaction(ctx, tx)
	if errSim != nil {
		return fmt.Errorf("transaction simulation failed: %v", err)
	}
	if simResult.Value.Err != nil {
		return fmt.Errorf("simulation error: %v", simResult.Value.Err)
	}
	log.Println("Transaction simulation succeeded.")

	// Send the transaction with confirmation (with a timeout).
	opts := rpc.TransactionOpts{
		SkipPreflight:       true,
		PreflightCommitment: rpc.CommitmentFinalized,
	}
	timeout := 10 * time.Second
	sig, errSend := confirm.SendAndConfirmTransactionWithOpts(ctx, rpcClient, wsClient, tx, opts, &timeout)
	//log.Println(errSend)
	if errSend != nil && errSend.Error() != "timeout" {
		log.Println(errSend)
		return fmt.Errorf("failed to send transaction: %v", errSend)
	}

	spew.Dump(tx)
	tx.EncodeTree(text.NewTreeEncoder(log.Writer(), "Buy Token"))
	log.Println("Transaction confirmed. Signature:", sig)
	return nil
}

func createAssociatedAccountWithRetry(payer solana.PrivateKey, mint solana.PublicKey, rpcClient *rpc.Client, wsClient *ws.Client, associatedTokenAddress solana.PublicKey) error {

	var err error
	// Retry up to maxRetries times
	for attempt := 0; attempt < MaxRetries; attempt++ {
		err = createAssociateAccount(payer, mint, rpcClient, wsClient)
		if err == nil {
			_, errAssociated := rpcClient.GetAccountInfo(context.TODO(), associatedTokenAddress)
			if errAssociated != nil {
				if strings.Contains(errAssociated.Error(), "not found") {
					// Log the error and retry
					fmt.Printf("Attempt %d failed: %v\n", attempt+1, err)
				}
			} else {
				return nil // Success
			}
		}
		// Exponential backoff (2^attempt seconds)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return fmt.Errorf("failed to create associated account after %d retries: %v", MaxRetries, err)
}
func createAssociateAccount(payer solana.PrivateKey, mint solana.PublicKey, rpcClient *rpc.Client, wsClient *ws.Client) error {
	ctx := context.TODO()
	log.Println("Creating Associated account")

	ata := associatedtokenaccount.NewCreateInstruction(
		payer.PublicKey(),
		payer.PublicKey(),
		mint,
	).Build()

	recentBlockhash, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("failed to fetch blockhash: %v", err)
	}

	priorityIx, errPriority := computebudget.NewSetComputeUnitPriceInstruction(PriorityFeeLamport).ValidateAndBuild()
	if errPriority != nil {
		return fmt.Errorf("failed to set priority: %v", errPriority)
	}
	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			priorityIx,
			ata,
		},
		recentBlockhash.Value.Blockhash,
		solana.TransactionPayer(payer.PublicKey()),
	)
	if err != nil {
		return fmt.Errorf("failed to create ATA transaction: %v", err)
	}
	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			if payer.PublicKey().Equals(key) {
				return &payer
			}
			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("unable to sign transaction:: %w", err)
	}
	spew.Dump(tx)

	simResult, errSim := rpcClient.SimulateTransaction(ctx, tx)
	if errSim != nil {
		return fmt.Errorf("transaction simulation failed: %v", err)
	}
	if simResult.Value.Err != nil {
		return fmt.Errorf("simulation error: %v", simResult.Value.Err)
	}
	log.Println("Transaction simulation succeeded.")

	// Pretty print the transaction:
	tx.EncodeTree(text.NewTreeEncoder(os.Stdout, "Create Associate Account"))

	opts := rpc.TransactionOpts{
		SkipPreflight:       true,
		PreflightCommitment: rpc.CommitmentFinalized,
	}
	t := 10 * time.Second // random default timeout
	timeout := &t
	sig, errSend := confirm.SendAndConfirmTransactionWithOpts(
		context.TODO(),
		rpcClient,
		wsClient,
		tx,
		opts,
		timeout,
	)
	//log.Println(errSend)
	if errSend != nil && errSend.Error() != "timeout" {
		log.Println(errSend)
		return fmt.Errorf("failed to send transaction: %v", errSend)
	}

	log.Println("Assoicate Account Created.")
	spew.Dump(sig)
	return nil
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
