package models

import (
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
)

type AccountInfo struct {
	Balance      uint64
	FinalBalance uint64
	Data         interface{}
	Tokens       map[solana.PublicKey]TokenInfo
}

type TokenInfo struct {
	AssociateAccount solana.PublicKey
	Token            token.Account
}
