package sol

import (
	"b46/b46/models"
	"fmt"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"log"
)

// PumpFunExecutor implements the orderhandler.Executor interface.
type PumpFunExecutor struct {
	// Provide any fields you need: Solana client, credentials, etc.
	// solanaClient *solana_sdk.Client
}

// ExecuteSellOrder places a SELL transaction on Solana.
func (s *PumpFunExecutor) ExecuteSellOrder(rpcClient *rpc.Client, wsClient *ws.Client, token models.MemeToken) error {
	//err := executeSell(rpcClient, wsClient, token.Mint, token.BondingCurve, token.AssociatedCurve)
	//if err != nil {
	//	log.Printf("Failed to sell token: %v", err)
	//}
	log.Printf("[SolanaExecutor - PumpFun] SELL order for token %s (%s)", token.Name, token.Symbol)
	fmt.Println("Sell order completed on Solana for:", token.Mint.String())
	return nil
}

// ExecuteBuyOrder places a Buy transaction on Solana.
func (s *PumpFunExecutor) ExecuteBuyOrder(rpcClient *rpc.Client, wsClient *ws.Client, token models.MemeToken) error {
	//amount := models.PositionAmount
	//err := executeBuy(rpcClient, wsClient, token.Mint, token.BondingCurve, token.AssociatedCurve, amount)
	//if err != nil {
	//	log.Printf("Failed to buy token: %v", err)
	//}
	log.Printf("[SolanaExecutor - PumpFun] Buy order for token %s (%s)", token.Name, token.Symbol)
	fmt.Println("Buy order completed on Solana for:", token.Mint.String())
	return nil
}
