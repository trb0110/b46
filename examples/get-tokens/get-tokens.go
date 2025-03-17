package main

import (
	"b46/b46/_sys_init"
	"context"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"math/big"
)

func main() {
	_ = _sys_init.NewEnviroSetup()

	endpoint := _sys_init.Env.RPC
	client := rpc.New(endpoint)
	// Decode the private key and create the payer
	payer := solana.MustPrivateKeyFromBase58(_sys_init.Env.PK)
	public := payer.PublicKey()

	out, err := client.GetTokenAccountsByOwner(
		context.TODO(),
		public,
		&rpc.GetTokenAccountsConfig{
			ProgramId: &solana.TokenProgramID,
		},
		&rpc.GetTokenAccountsOpts{
			Encoding: solana.EncodingBase64Zstd,
		},
	)
	if err != nil {
		panic(err)
	}
	spew.Dump(out)

	{
		tokenAccounts := make([]token.Account, 0)
		for _, rawAccount := range out.Value {
			var tokAcc token.Account

			data := rawAccount.Account.Data.GetBinary()
			dec := bin.NewBinDecoder(data)
			err := dec.Decode(&tokAcc)
			if err != nil {
				panic(err)
			}
			tokenAccounts = append(tokenAccounts, tokAcc)
		}
		spew.Dump(tokenAccounts)
	}
}

// getTokenBalance fetches the balance of an SPL token account
func getTokenBalance(associatedTokenAccount string) (string, error) {
	client := rpc.New(_sys_init.Env.RPC)

	// Convert to Solana PublicKey
	accountPubKey, err := solana.PublicKeyFromBase58(associatedTokenAccount)
	if err != nil {
		return "", fmt.Errorf("invalid account address: %w", err)
	}

	// Fetch token account balance
	resp, err := client.GetTokenAccountBalance(context.TODO(), accountPubKey, rpc.CommitmentConfirmed)
	if err != nil {
		return "", fmt.Errorf("failed to get token balance: %w", err)
	}

	// Convert balance to uint64
	if resp != nil && resp.Value != nil {
		return resp.Value.Amount, nil
	}

	return "", nil // Return 0 if no balance
}

// getPumpFunTokenBalance fetches the balance of an SPL token account
func getPumpFunTokenBalance(walletAddress string, tokenMint string) (string, error) {
	client := rpc.New(_sys_init.Env.RPC)

	// Convert to Solana PublicKeys
	walletPubKey, err := solana.PublicKeyFromBase58(walletAddress)
	if err != nil {
		return "0", fmt.Errorf("invalid wallet address: %w", err)
	}
	tokenMintPubKey, err := solana.PublicKeyFromBase58(tokenMint)
	if err != nil {
		return "0", fmt.Errorf("invalid token mint address: %w", err)
	}

	// Derive Associated Token Account (ATA)
	associatedTokenAccount, _, err := solana.FindAssociatedTokenAddress(walletPubKey, tokenMintPubKey)
	if err != nil {
		return "0", fmt.Errorf("failed to derive ATA: %w", err)
	}

	// Fetch token account balance
	resp, err := client.GetTokenAccountBalance(context.TODO(), associatedTokenAccount, rpc.CommitmentFinalized)
	if err != nil {
		return "0", fmt.Errorf("failed to get token balance: %w", err)
	}

	// Convert balance to uint64
	if resp != nil && resp.Value != nil {
		return resp.Value.Amount, nil
	}

	return "0", nil // Return 0 if no balance
}
func GetBalance(address string) *big.Float {
	endpoint := _sys_init.Env.RPC
	client := rpc.New(endpoint)

	pubKey := solana.MustPublicKeyFromBase58(address)
	out, err := client.GetBalance(
		context.TODO(),
		pubKey,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		panic(err)
	}
	spew.Dump(out)
	spew.Dump(out.Value) // total lamports on the account; 1 sol = 1000000000 lamports

	var lamportsOnAccount = new(big.Float).SetUint64(uint64(out.Value))
	// Convert lamports to sol:
	var solBalance = new(big.Float).Quo(lamportsOnAccount, new(big.Float).SetUint64(solana.LAMPORTS_PER_SOL))
	return solBalance
}

func GetTokenInfo(address string) {
	endpoint := _sys_init.Env.RPC
	client := rpc.New(endpoint)

	pubKey := solana.MustPublicKeyFromBase58(address)

	{
		// basic usage
		resp, err := client.GetAccountInfo(
			context.TODO(),
			pubKey,
		)
		if err != nil {
			panic(err)
		}
		spew.Dump(resp)

		var mint token.Mint
		// Account{}.Data.GetBinary() returns the *decoded* binary data
		// regardless the original encoding (it can handle them all).
		err = bin.NewBinDecoder(resp.Value.Data.GetBinary()).Decode(&mint)
		if err != nil {
			panic(err)
		}
		spew.Dump(mint)
		// NOTE: The supply is mint.Supply, with the mint.Decimals:
		// mint.Supply = 9998022451607088
		// mint.Decimals = 6
		// ... which means that the supply is 9998022451.607088
	}
	{
		var mint token.Mint
		// Get the account, and decode its data into the provided mint object:
		err := client.GetAccountDataInto(
			context.TODO(),
			pubKey,
			&mint,
		)
		if err != nil {
			panic(err)
		}
		spew.Dump(mint)
	}
	{
		// advanced usage
		resp, err := client.GetAccountInfoWithOpts(
			context.TODO(),
			pubKey,
			// You can specify more options here:
			&rpc.GetAccountInfoOpts{
				Encoding:   solana.EncodingBase64Zstd,
				Commitment: rpc.CommitmentFinalized,
				// You can get just a part of the account data by specify a DataSlice:
				// DataSlice: &rpc.DataSlice{
				// 	Offset: pointer.ToUint64(0),
				// 	Length: pointer.ToUint64(1024),
				// },
			},
		)
		if err != nil {
			panic(err)
		}
		spew.Dump(resp)

		var mint token.Mint
		err = bin.NewBinDecoder(resp.Value.Data.GetBinary()).Decode(&mint)
		if err != nil {
			panic(err)
		}
		spew.Dump(mint)
	}
}
