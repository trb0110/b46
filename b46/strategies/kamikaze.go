package strategies

import (
	"b46/b46/_sys_init"
	analysis "b46/b46/chart-analysis"
	"b46/b46/logging"
	"b46/b46/models"
	"b46/b46/sol"
	"context"
	"github.com/coder/websocket"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"log"
	"strconv"
	"sync"
	"time"
)

type Kamikaze struct {
	Context context.Context
	sync.Mutex
	RpcClient *rpc.Client
	WssClient *ws.Client
	Websocket *websocket.Conn
}

func (kami *Kamikaze) InitializeKamikaze() {
	kami.Lock()
	kami.Context = context.Background()

	kami.RpcClient = rpc.New(_sys_init.Env.RPC)
	wssClient, errorWss := ws.Connect(kami.Context, _sys_init.Env.WSS)
	if errorWss != nil {
		logging.PrintErrorToLog("Failed to initialize wss client:		", errorWss.Error())
	}
	kami.WssClient = wssClient

	conn, _, err := websocket.Dial(kami.Context, _sys_init.Env.WSS, nil)
	if err != nil {
		logging.PrintErrorToLog("Failed to initialize websocket:		", err.Error())
	}
	kami.Websocket = conn

	defer kami.Unlock()
}

func (kami *Kamikaze) Start() {
	ctx, _ := context.WithCancel(context.Background())

	models.InitializePumpMemes()

	go kami.ListenPumpFun()
	go kami.MonitorMemes()

	// Instantiate the executor that knows how to do the actual sells.
	solanaExec := &sol.PumpFunExecutor{}
	orderHandler := NewTradeHandler(solanaExec, 100)

	go orderHandler.Start(ctx)

	kami.Trade(orderHandler)
}

func (kami *Kamikaze) ListenPumpFun() {
	memeData := make(chan models.MemeToken)
	go sol.PumpFunListener(kami.Websocket, memeData)

	go func(ch <-chan models.MemeToken) {
		for data := range ch { // Continuously receive from the channel
			//fmt.Printf("📥 Consumer: Received data %v\n", data)
			//log.Println(models.PumpMemes)
			finalMemeToken := kami.UpdateMemeToken(data, 0)
			sol.PrintTokenMeme(finalMemeToken)
			models.PumpMemes.SetToken(finalMemeToken)
		}
		defer close(memeData)
	}(memeData)
}

func (kami *Kamikaze) MonitorMemes() {
	// Poll the tokens every 20 seconds
	ticker := time.NewTicker(models.MONITOR_DURATION * time.Second)
	defer ticker.Stop()

	errInitLogger := logging.InitLogger("monitor.log")
	if errInitLogger != nil {
		logging.PrintErrorToLog("Error init logger:		", errInitLogger.Error())
	}
	defer func() {
		// Ensure we close the logger before exiting
		if cerr := logging.CloseLoggerFile("monitor.log"); cerr != nil {
			logging.PrintErrorToLog("Error close logger:		", cerr.Error())
		}
	}()
	for range ticker.C {
		tokens := models.PumpMemes.GetTokens()
		log.Println("###################################MONITOR##########################################")

		if err := logging.PrintToLog("monitor.log", []string{
			"#", "#", "#", "#", "#", "#", "#", "#", "#", "#", "#", "#", "#",
		}); err != nil {
			// Optional local error handling
			logging.PrintErrorToLog("logger write error:			", err.Error())
		}

		// The map you get here is a copy (if you coded GetTokens that way),
		// so it’s safe to range over
		for key, token := range tokens {
			index := len(token.Info)
			updatedMemeToken := kami.UpdateMemeToken(token, index)
			models.PumpMemes.SetToken(updatedMemeToken)

			//log.Println(key, updatedMemeToken)
			if err := logging.PrintToLog("monitor.log", []string{
				"MONITOR", key, updatedMemeToken.Name, updatedMemeToken.Symbol, updatedMemeToken.AddedTime.String(), logging.MemeInfosToString(updatedMemeToken.Info), strconv.FormatBool(updatedMemeToken.Trading),
			}); err != nil {
				// Optional local error handling
				logging.PrintErrorToLog("logger write error:			", err.Error())
			}

			tokenHistoryLength := len(updatedMemeToken.Info)
			finalMarketCap := updatedMemeToken.Info[len(updatedMemeToken.Info)-1].MarketCap
			if tokenHistoryLength > models.MaxEntryHistory && finalMarketCap < models.EntryMarketCap {
				log.Println("REMOVE				:", updatedMemeToken.Mint.String())
				if err := logging.PrintToLog("monitor.log", []string{
					"REMOVE", key, updatedMemeToken.Name, updatedMemeToken.Symbol, updatedMemeToken.AddedTime.String(), logging.MemeInfosToString(updatedMemeToken.Info), strconv.FormatBool(updatedMemeToken.Trading),
				}); err != nil {
					// Optional local error handling
					logging.PrintErrorToLog("logger write error:			", err.Error())
				}
				models.PumpMemes.DeleteToken(updatedMemeToken.Mint.String())
			}
			if tokenHistoryLength > models.MinEntryHistory && finalMarketCap > models.EntryMarketCap && updatedMemeToken.Trading == false {
				log.Println("ADD TO TRADES		:", updatedMemeToken.Mint.String())

				//ADD TO TRADES MAP
				models.TradesMap.SetToken(updatedMemeToken)

				updatedMemeToken.Trading = true
				models.PumpMemes.SetToken(updatedMemeToken)

				if err := logging.PrintToLog("monitor.log", []string{
					"ADD", key, updatedMemeToken.Name, updatedMemeToken.Symbol, updatedMemeToken.AddedTime.String(), logging.MemeInfosToString(updatedMemeToken.Info), strconv.FormatBool(updatedMemeToken.Trading),
				}); err != nil {
					// Optional local error handling
					logging.PrintErrorToLog("logger write error:			", err.Error())
				}

			}
		}
		// Optionally flush:
		if err := logging.FlushLog("monitor.log"); err != nil {
			logging.PrintErrorToLog("logger flush error:			", err.Error())
		}
	}
}

func (kami *Kamikaze) UpdateMemeToken(data models.MemeToken, index int) models.MemeToken {

	curveState, err := sol.GetPumpCurveState(kami.RpcClient, data.BondingCurve)
	if err != nil {
		logging.PrintErrorToLog("failed to fetch bonding curve state:		", err.Error())
	}
	t := time.Now()
	memeInfo := models.MemeInfo{curveState, 0, 0, t}
	data.Info = append(data.Info, memeInfo)
	if data.Info[index].BondingState != nil {
		// (2) Calculate token price and amount.
		tokenPrice, err := sol.CalculatePumpCurvePrice(curveState)
		if err != nil {
			logging.PrintErrorToLog("failed to calculate pump curve price:		", err.Error())
		}
		marketCap := sol.GetTokenMarketCap(data.Info[index].BondingState, tokenPrice)
		data.Info[index].TokenPrice = tokenPrice
		data.Info[index].MarketCap = marketCap
	}

	data.Migrated = false
	return data

}

func (kami *Kamikaze) Trade(trader *Trader) {

	//Poll the tokens every 20 seconds
	ticker := time.NewTicker(models.MONITOR_DURATION_TRADES * time.Second)
	defer ticker.Stop()

	errInitLogger := logging.InitLogger("trade.log")
	if errInitLogger != nil {
		logging.PrintErrorToLog("Error init logger:		", errInitLogger.Error())
	}
	defer func() {
		// Ensure we close the logger before exiting
		if cerr := logging.CloseLoggerFile("trade.log"); cerr != nil {
			logging.PrintErrorToLog("Error close logger:		", cerr.Error())
		}
	}()

	for range ticker.C {
		//if err := logging.ClearFileLog("trade.log"); err != nil {
		//	logging.PrintErrorToLog("Error clearing file:		", err.Error())
		//}
		tokens := models.TradesMap.GetTokens()
		log.Println("###################################TRADING##########################################")

		if err := logging.PrintToLog("trade.log", []string{
			"#", "#", "#", "#", "#", "#", "#", "#", "#", "#", "#", "#", "#",
		}); err != nil {
			// Optional local error handling
			logging.PrintErrorToLog("logger write error:			", err.Error())
		}

		// The map you get here is a copy (if you coded GetTokens that way),
		// so it’s safe to range over
		for key, token := range tokens {
			index := len(token.Info)
			updatedMemeToken := kami.UpdateMemeToken(token, index)

			tokenAnalysis := analysis.AnalyzeTokenData(updatedMemeToken)
			updatedMemeToken.Analysis = append(updatedMemeToken.Analysis, tokenAnalysis)

			//Update in trades map
			models.TradesMap.SetToken(updatedMemeToken)
			tokenHistoryLength := len(updatedMemeToken.Info)
			finalMarketCap := updatedMemeToken.Info[tokenHistoryLength-1].MarketCap

			//log.Println(key, updatedMemeToken)
			if err := logging.PrintToLog("trade.log", []string{
				"CURRENTLY TRADING", key, updatedMemeToken.Name, updatedMemeToken.Symbol, updatedMemeToken.AddedTime.String(), logging.MemeInfosToString(updatedMemeToken.Info), logging.AnalysisInfosToString(updatedMemeToken.Analysis), strconv.FormatBool(updatedMemeToken.Trading), strconv.FormatBool(updatedMemeToken.Sold),
			}); err != nil {
				// Optional local error handling
				logging.PrintErrorToLog("logger write error:			", err.Error())
			}

			if updatedMemeToken.Trading == false {
				trader.SubmitOrder(OrderRequest{
					Token:     updatedMemeToken,
					OrderType: OrderTypeBuy,
					Reason:    "Entry market cap",
				})
			}

			if updatedMemeToken.Trading == true && updatedMemeToken.Sold == false {
				if finalMarketCap > models.ExitMarketCap {
					log.Println("REMOVE FROM TRADING		:", updatedMemeToken.Mint.String())
					trader.SubmitOrder(OrderRequest{
						Token:     updatedMemeToken,
						OrderType: OrderTypeSell,
						Reason:    "Above exit market cap",
					})
				}
			}
		}
		// Optionally flush:
		if err := logging.FlushLog("trade.log"); err != nil {
			logging.PrintErrorToLog("logger flush error:			", err.Error())
		}
	}
}
