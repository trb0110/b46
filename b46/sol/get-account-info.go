package sol

import (
	"b46/b46/_sys_init"
	"b46/b46/models"
	"context"
	"fmt"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"log"
)

// FindAssociatedBondingCurve derives the associated bonding curve address (using the ATA derivation)
// from the bonding curve and the mint address. The seeds used here are:
// bondingCurve, token program, and mint.
func GetAccountInfo(rpcClient *rpc.Client) models.AccountInfo {
	payer := solana.MustPrivateKeyFromBase58(_sys_init.Env.PK)
	account := models.AccountInfo{
		Tokens: make(map[solana.PublicKey]models.TokenInfo), // 🔹 Initialize the map
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
	//fmt.Println("Account 		:", account)

	// 3) Convert lamports -> SOL
	//initialSolBalance := float64(account.Balance) / 1e9
	totalSOL := float64(account.Balance) / 1e9
	//fmt.Printf("Initial SOL Balance: %.20f SOL\n", initialSolBalance)

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
		var tokenAccount models.TokenInfo
		var tokAcc token.Account

		//log.Println(rawAccount)
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

	// 6) Now convert each token to SOL
	//    We must fetch:
	//      a) the token mint’s decimals
	//      b) the price in SOL (or fetch price in USD and convert USD->SOL).
	//for _, tk := range account.Tokens {
	//	mint := tk.Token.Mint
	//	// Derive the bonding curve address (and get the bump seed).
	//	bondingCurveAddress, bump, _ := GetBondingCurveAddress(mint, models.PumpProgramPublic)
	//
	//	// Calculate the associated bonding curve address.
	//	associatedBondingCurve := FindAssociatedBondingCurve(mint, bondingCurveAddress)
	//
	//	fmt.Println(":")
	//	fmt.Println("------------------ACCOUNT INFO----------------------")
	//	fmt.Printf("Token Mint:               %s", mint.String())
	//	fmt.Printf("Bonding Curve:            %s", bondingCurveAddress.String())
	//	fmt.Printf("Associated Bonding Curve: %s", associatedBondingCurve.String())
	//	fmt.Printf("Bonding Curve Bump:       %d", bump)
	//	fmt.Printf("Amount Held:       %d", tk.Token.Amount)
	//
	//	curveState, err := GetPumpCurveState(rpcClient, bondingCurveAddress)
	//	if err != nil {
	//		logging.PrintErrorToLog("failed to fetch bonding curve state:		", err.Error())
	//	}
	//	fmt.Println("Curve State:		", curveState)
	//	tokenPrice, err := CalculatePumpCurvePrice(curveState)
	//	if err != nil {
	//		logging.PrintErrorToLog("failed to calculate pump curve price:		", err.Error())
	//	}
	//
	//	fmt.Println("--------------------------------------------------")
	//	totalSOL += tokenPrice
	//	//fmt.Printf("Token mint %v => %f tokens => %f SOL\n", mint, quantity, tokenValueInSOL)
	//}
	account.FinalBalance = uint64(totalSOL)

	return account
}
