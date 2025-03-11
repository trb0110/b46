package main

import (
	"b46/b46/_sys_init"
	"context"
	"fmt"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"log"
)

type AccountInfo struct {
	Balance uint64
	Data    interface{}
	Tokens  []TokenInfo
}

type TokenInfo struct {
	AssociateAccount solana.PublicKey
	Token            token.Account
}

func main() {
	_ = _sys_init.NewEnviroSetup()

	rpcClient := rpc.New(_sys_init.Env.RPC)
	payer := solana.MustPrivateKeyFromBase58(_sys_init.Env.PK)

	var account AccountInfo
	accountInfo, _ := rpcClient.GetAccountInfo(context.TODO(), payer.PublicKey())
	fmt.Println("Account Info		:", accountInfo.Value)

	accountInfoData := accountInfo.Value.Data.GetBinary()
	dec := bin.NewBinDecoder(accountInfoData)
	err := dec.Decode(&account.Data)
	if err != nil {
		log.Println("fail to decode account info", err)
	}

	account.Balance = accountInfo.Value.Lamports
	fmt.Println("Account 		:", account)

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
		log.Println(tokAcc)

		tokenAccount.AssociateAccount = rawAccount.Pubkey
		tokenAccount.Token = tokAcc
		account.Tokens = append(account.Tokens, tokenAccount)
	}

	log.Println(account)

}
