package models

import (
	"fmt"
	"github.com/gagliardetto/solana-go"
)

var (
	PumpProgramPublic                   = solana.MustPublicKeyFromBase58("6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P")
	PumpGlobal                          = solana.MustPublicKeyFromBase58("4wTV1YmiEkRvAtNtsSGPtUrqRYQMe5SKy2uB4Jjaxnjf")
	PumpFee                             = solana.MustPublicKeyFromBase58("CebN5WGQ4jvEPvsVU4EoHEpgzq1VV7AbicfhtW4xC9iM")
	SystemProgram                       = solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	SystemTokenProgram                  = solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	SystemAssociatedTokenAccountProgram = solana.MustPublicKeyFromBase58("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL")
	SystemRent                          = solana.MustPublicKeyFromBase58("SysvarRent111111111111111111111111111111111")
	PumpEventAuthority                  = solana.MustPublicKeyFromBase58("Ce6TQqeHC9p8KetsN6JsjHK7UTZk7nasjjnr7XxXp9F1")
)

func (b *BondingCurveState) String() string {
	if b == nil {
		return "nil"
	}
	return fmt.Sprintf("BondingCurveState{CurrentSupply: %d, VirtualTokenReserves: %d,  RealTokenReserves: %d,  VirtualSolReserves: %d,  RealSolReserves: %d}", b.TokenTotalSupply, b.VirtualTokenReserves, b.RealTokenReserves, b.VirtualSolReserves, b.RealSolReserves)
}

// BondingCurveState represents the state of a bonding curve
type BondingCurveState struct {
	VirtualTokenReserves uint64
	VirtualSolReserves   uint64
	RealTokenReserves    uint64
	RealSolReserves      uint64
	TokenTotalSupply     uint64
	Complete             bool
}

//1000000000 000000

type PumpFun struct {
	Version      string `json:"version"`
	Name         string `json:"name"`
	Instructions []struct {
		Name     string   `json:"name"`
		Docs     []string `json:"docs"`
		Accounts []struct {
			Name     string `json:"name"`
			IsMut    bool   `json:"isMut"`
			IsSigner bool   `json:"isSigner"`
		} `json:"accounts"`
		Args []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"args"`
	} `json:"instructions"`
	Accounts []struct {
		Name string `json:"name"`
		Type struct {
			Kind   string `json:"kind"`
			Fields []struct {
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"fields"`
		} `json:"type"`
	} `json:"accounts"`
	Events []struct {
		Name   string `json:"name"`
		Fields []struct {
			Name  string `json:"name"`
			Type  string `json:"type"`
			Index bool   `json:"index"`
		} `json:"fields"`
	} `json:"events"`
	Errors []struct {
		Code int    `json:"code"`
		Name string `json:"name"`
		Msg  string `json:"msg"`
	} `json:"errors"`
	Metadata struct {
		Address string `json:"address"`
	} `json:"metadata"`
}
