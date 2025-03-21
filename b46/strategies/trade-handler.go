package strategies

import (
	"b46/b46/_sys_init"
	"b46/b46/helpers"
	"b46/b46/logging"
	"b46/b46/models"
	"context"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"log"
)

// OrderType enumerates the possible order actions.
type OrderType int

const (
	OrderTypeSell OrderType = iota
	OrderTypeBuy  OrderType = iota
)

type OrderResponse struct {
	Success bool
	Error   string
}

// OrderRequest encapsulates all data required to execute an order.
type OrderRequest struct {
	Token      models.MemeToken
	OrderType  OrderType
	Reason     string
	ResultChan chan OrderResponse
}

// Executor is an interface that the order handler can call to execute a particular order.
// This decouples the handler's concurrency logic from the actual trading implementation.
type Executor interface {
	ExecuteSellOrder(rpcClient *rpc.Client, wsClient *ws.Client, token models.MemeToken) error
	ExecuteBuyOrder(rpcClient *rpc.Client, wsClient *ws.Client, token models.MemeToken) error
	// Add more methods if you have other order types.
}

// Handler is responsible for concurrently processing all incoming orders.
type Trader struct {

	// Buffered channel for inbound orders.
	orderChannel chan OrderRequest
	executor     Executor
	RpcClient    *rpc.Client
	WssClient    *ws.Client
}

// NewHandler instantiates a new order handler with a given executor.
// bufferSize is the size of the channel buffer for incoming orders.
func NewTradeHandler(executor Executor, bufferSize int) *Trader {
	wssClient, errorWss := ws.Connect(context.Background(), _sys_init.Env.WSS)
	if errorWss != nil {
		logging.PrintErrorToLog("Failed to initialize wss client:		", errorWss.Error())
	}
	return &Trader{
		orderChannel: make(chan OrderRequest, bufferSize),
		executor:     executor,
		RpcClient:    rpc.New(_sys_init.Env.RPC),
		WssClient:    wssClient,
	}
}

func (t *Trader) Start(ctx context.Context) {
	errInitLogger := logging.InitLogger("trades.log")
	if errInitLogger != nil {
		logging.PrintErrorToLog("Error init trades logger:", errInitLogger.Error())
	}
	defer func() {
		if cerr := logging.CloseLoggerFile("trades.log"); cerr != nil {
			logging.PrintErrorToLog("Error close trades logger:", cerr.Error())
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("OrderHandler: context canceled, shutting down.")
			return
		case orderReq := <-t.orderChannel:
			// Process each order concurrently.
			go t.handleOrder(orderReq)
		}
	}
}

func (t *Trader) handleOrder(orderReq OrderRequest) {
	//account := sol.GetAccountInfo(t.RpcClient)
	tokenHistoryLength := len(orderReq.Token.Info)
	finalMarketCap := orderReq.Token.Info[tokenHistoryLength-1].MarketCap
	finalPrice := orderReq.Token.Info[tokenHistoryLength-1].TokenPrice
	switch orderReq.OrderType {
	case OrderTypeSell:

		//models.TradesMap.DeleteToken(orderReq.Token.Mint.String())

		orderReq.Token.Sold = true
		models.TradesMap.SetToken(orderReq.Token)
		if err := logging.PrintToLog("trades.log", []string{
			"SELL", orderReq.Token.Mint.String(), orderReq.Token.Name, orderReq.Token.Symbol, helpers.ConvertFloatToString(finalMarketCap), helpers.ConvertFloatToString(finalPrice), orderReq.Reason,
		}); err != nil {
			logging.PrintErrorToLog("logger write error:", err.Error())
		}
		err := t.executor.ExecuteSellOrder(t.RpcClient, t.WssClient, orderReq.Token)
		if err != nil {
			log.Printf("Sell order failed: token=%s err=%v\n", orderReq.Token.Mint.String(), err)
		} else {
			log.Printf("Sell order executed for token=%s reason=%s\n", orderReq.Token.Mint.String(), orderReq.Reason)
		}
	case OrderTypeBuy:

		orderReq.Token.Trading = true
		models.TradesMap.SetToken(orderReq.Token)
		if err := logging.PrintToLog("trades.log", []string{
			"BUY", orderReq.Token.Mint.String(), orderReq.Token.Name, orderReq.Token.Symbol, helpers.ConvertFloatToString(finalMarketCap), helpers.ConvertFloatToString(finalPrice), orderReq.Reason,
		}); err != nil {
			logging.PrintErrorToLog("logger write error:", err.Error())
		}
		err := t.executor.ExecuteBuyOrder(t.RpcClient, t.WssClient, orderReq.Token)
		if err != nil {
			log.Printf("Buy order failed: token=%s err=%v\n", orderReq.Token.Mint.String(), err)
		} else {
			log.Printf("Buy order executed for token=%s reason=%s\n", orderReq.Token.Mint.String(), orderReq.Reason)
		}
	default:
		log.Printf("Unknown OrderType=%d\n", orderReq.OrderType)
	}
}

// SubmitOrder is used by external code to send new orders into the handler.
func (t *Trader) SubmitOrder(req OrderRequest) {
	t.orderChannel <- req
}
