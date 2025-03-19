package sol

import (
	"b46/b46/_sys_init"
	"b46/b46/models"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/gagliardetto/solana-go"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/rpc"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/gagliardetto/solana-go/text"
	"log"
	"math"
	"os"
	"time"
)

func executeSell(rpcClient *rpc.Client, wsClient *ws.Client, mint, bondingCurve, associatedBondingCurve solana.PublicKey) error {

	ctx := context.Background()

	// Decode the private key and create the payer
	payer := solana.MustPrivateKeyFromBase58(_sys_init.Env.PK)

	//public := payer.PublicKey()
	//log.Println(public)

	// Derive the associated token address
	associatedTokenAddress, _, _ := solana.FindAssociatedTokenAddress(
		payer.PublicKey(),
		mint,
	)
	//log.Println(associatedTokenAddress)
	balance, err := GetTokenBalance(ctx, rpcClient, associatedTokenAddress)
	if err != nil {
		fmt.Printf("Failed to get token balance: %v\n", err)

	}

	// Convert to a decimal amount (i.e. human-readable token balance)
	tokenBalanceDecimal := float64(balance) / math.Pow10(models.TOKEN_DECIMALS)
	//fmt.Printf("Token balance: %f\n", tokenBalanceDecimal)
	if balance == 0 {
		fmt.Println("No tokens to sell.")

	}
	// Fetch bonding curve state (assumed implemented as getPumpCurveState)
	curveState, err := GetPumpCurveState(rpcClient, bondingCurve)
	if err != nil {
		return fmt.Errorf("failed to fetch bonding curve state: %v", err)
	}

	tokenPrice, err := CalculatePumpCurvePrice(curveState)
	if err != nil {
		return fmt.Errorf("calculate pump curve: %v", err)
	}

	//fmt.Printf("Token Balance: %f\n", tokenBalanceDecimal)
	//log.Println("Token price:	", tokenPrice)
	//log.Println("Curve State:	", curveState)
	//// Compute Associated Token Address

	// === Calculate minimum SOL output ===
	// Compute the minimum SOL (in lamports) you would accept.
	// First, compute the output in SOL as a float.
	amount := balance
	minSolOutputFloat := tokenBalanceDecimal * tokenPrice
	slippageFactor := 1 - models.Slippage
	// Multiply by the slippage factor and convert SOL to lamports (and then to an integer).
	minSolOutput := float64((minSolOutputFloat * slippageFactor) * models.LamportsPerSOL)

	//fmt.Printf("Selling %f tokens\n", tokenBalanceDecimal)
	//fmt.Printf("Minimum SOL output: %.10f SOL\n", float64(minSolOutput)/models.LamportsPerSOL)

	time.Sleep(3 * time.Second)
	//Create and send the buy transaction
	errBuy := sellTokenWithRetry(payer, mint, rpcClient, wsClient, associatedTokenAddress, bondingCurve, associatedBondingCurve, amount, minSolOutput)
	if errBuy != nil {
		log.Printf("Failed to create associated token account: %v", errBuy)
	}

	return nil
}

func sellTokenWithRetry(payer solana.PrivateKey, mint solana.PublicKey, rpcClient *rpc.Client, wsClient *ws.Client, associatedTokenAddress, bondingCurve, associatedBondingCurve solana.PublicKey, tokenAmount int, minSolOutput float64) error {

	var err error
	// Retry up to maxRetries times
	for attempt := 0; attempt < models.MaxRetries; attempt++ {
		err = sellToken(payer, mint, rpcClient, wsClient, associatedTokenAddress, bondingCurve, associatedBondingCurve, tokenAmount, minSolOutput)
		if err == nil {
			return nil
		}
		// Exponential backoff (2^attempt seconds)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return fmt.Errorf("failed to create associated account after %d retries: %v", models.MaxRetries, err)
}

func sellToken(payer solana.PrivateKey, mint solana.PublicKey, rpcClient *rpc.Client, wsClient *ws.Client, associatedTokenAddress, bondingCurve, associatedBondingCurve solana.PublicKey, tokenAmount int, minSolOutput float64) error {

	data := make([]byte, 24)

	binary.LittleEndian.PutUint64(data[0:], 12502976635542562355)
	// Write the amount (little-endian uint64) at offset 8.
	binary.LittleEndian.PutUint64(data[8:], uint64(tokenAmount))
	// Write the min_sol_output (little-endian uint64) at offset 16.
	binary.LittleEndian.PutUint64(data[16:], uint64(minSolOutput))
	//
	//log.Println(tokenAmount)
	//log.Println(minSolOutput)

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

	// Add a compute budget instruction to set a higher compute unit price (priority fee).
	// This helps ensure your transaction is processed faster in a congested network.
	priorityIx, errPriority := computebudget.NewSetComputeUnitPriceInstruction(models.PriorityFeeLamport).ValidateAndBuild()
	if errPriority != nil {
		return fmt.Errorf("failed to set priority: %v", errPriority)
	}
	recentBlockhash, err := rpcClient.GetLatestBlockhash(context.TODO(), rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("failed to fetch blockhash: %v", err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			priorityIx,      // Priority fee instruction
			sellInstruction, // Main buy instruction
		},
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
	//spew.Dump(tx)
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
	//spew.Dump(sig)
	log.Println("Transaction confirmed. Signature:", sig)

	//log.Println("Transaction confirmed.")
	return nil
}
