package models

import (
	"fmt"
	"github.com/gagliardetto/solana-go"
	"sync"
	"time"
)

type MemeToken struct {
	Name            string
	Symbol          string
	URI             string
	Mint            solana.PublicKey
	BondingCurve    solana.PublicKey
	User            string
	AssociatedCurve solana.PublicKey
	AddedTime       time.Time
	Info            []MemeInfo
	Analysis        []TokenAnalysis
	Migrated        bool
	Trading         bool
	Sold            bool
}
type MemeInfo struct {
	BondingState *BondingCurveState
	TokenPrice   float64
	MarketCap    float64
	Snapshot     time.Time
}

// Analysis contains the results of our evaluation of a token's state.
type TokenAnalysis struct {
	// Basic liquidity & valuation criteria:
	MarketCapSufficiency bool // true if the token's market cap meets or exceeds a threshold
	ReservesSufficiency  bool // true if the ratio of reserves to supply meets a minimum threshold
	DataPoints           int  // Number of snapshots analyzed

	// Liquidity/reserve indicators:
	ReserveRatio    float64 // RealTokenReserves / VirtualTokenReserves
	SolReserveRatio float64 // RealSolReserves / VirtualSolReserves (if applicable)

	// Price behavior over time (across snapshots):
	PriceStability         bool    // true if the variance of token price is below a given maximum
	PriceConvergence       float64 // true if the range (max - min) of token prices is within an acceptable delta
	SimpleMovingAverage    float64 // the average token price over the analysis period
	PercentageChange       float64 // percentage change from the first to last snapshot
	PriceTrendSlope        float64 // slope from a linear regression on token price snapshots
	ConsistentlyTrendingUp bool    // true if each snapshot shows an increase compared to the previous one

	// Risk / reward metrics:
	Volatility       float64 // standard deviation of the token price over the analysis period
	TheoreticalPrice float64 // computed theoretical price from the bonding curve data
	CurrentPrice     float64 // current observed token price (or last snapshot price)
	RiskRewardScore  float64 // a normalized score (e.g., discount divided by volatility)
	Snapshot         time.Time
}

type Meme_Sync struct {
	sync.Mutex
	Tokens map[string]MemeToken
}
type Trades_Sync struct {
	sync.Mutex
	Tokens map[string]MemeToken
}

var PumpMemes Meme_Sync
var TradesMap Trades_Sync

func InitializePumpMemes() {
	PumpMemes = Meme_Sync{
		Tokens: make(map[string]MemeToken),
	}
	TradesMap = Trades_Sync{
		Tokens: make(map[string]MemeToken),
	}
}

func (post *Meme_Sync) Get(key string) (MemeToken, bool) {
	post.Lock()
	defer post.Unlock()
	tok, exists := post.Tokens[key]
	if !exists {
		return MemeToken{}, false
	}
	return tok, exists
}

func (post *Meme_Sync) GetTokens() map[string]MemeToken {
	post.Lock()
	defer post.Unlock()
	// If you return p.tokens directly, any code that iterates over that map
	// could still race if there's writing happening concurrently.
	// It's often safest to return a COPY of the map:
	copyMap := make(map[string]MemeToken, len(post.Tokens))
	for k, v := range post.Tokens {
		copyMap[k] = deepCopyMemeToken(v)
	}
	return copyMap
}
func (post *Meme_Sync) CountTokens() int {
	post.Lock()
	defer post.Unlock()
	return len(post.Tokens)
}

func (post *Meme_Sync) SetToken(meme MemeToken) {
	post.Lock()
	defer post.Unlock()
	t := time.Now()
	meme.AddedTime = t

	post.Tokens[meme.Mint.String()] = meme
}

func (post *Meme_Sync) DeleteAllTokens() {
	post.Lock()
	defer post.Unlock()
	for key, _ := range post.Tokens {
		delete(post.Tokens, key)
	}
}
func (post *Meme_Sync) DeleteToken(key string) {
	post.Lock()
	defer post.Unlock()
	delete(post.Tokens, key)
}

func (post *Trades_Sync) Get(key string) (MemeToken, bool) {
	post.Lock()
	defer post.Unlock()
	tok, exists := post.Tokens[key]
	if !exists {
		return MemeToken{}, false
	}
	return tok, exists
}

func (post *Trades_Sync) GetTokens() map[string]MemeToken {
	post.Lock()
	defer post.Unlock()

	// If you return p.tokens directly, any code that iterates over that map
	// could still race if there's writing happening concurrently.
	// It's often safest to return a COPY of the map:
	copyMap := make(map[string]MemeToken, len(post.Tokens))
	for k, v := range post.Tokens {
		copyMap[k] = deepCopyMemeToken(v)
	}
	return copyMap
}

func (post *Trades_Sync) CountTokens() int {
	post.Lock()
	defer post.Unlock()
	return len(post.Tokens)
}

func (post *Trades_Sync) SetToken(meme MemeToken) {
	post.Lock()
	defer post.Unlock()
	t := time.Now()
	meme.AddedTime = t

	post.Tokens[meme.Mint.String()] = meme
}

func (post *Trades_Sync) DeleteAllTokens() {
	post.Lock()
	defer post.Unlock()
	for key, _ := range post.Tokens {
		delete(post.Tokens, key)
	}
}
func (post *Trades_Sync) DeleteToken(key string) {
	post.Lock()
	defer post.Unlock()
	delete(post.Tokens, key)
}

func (m MemeInfo) String() string {
	return fmt.Sprintf(
		"MemeInfo{BondingState: %v, TokenPrice: %.20f, MarketCap: %.2f, Time: %s}",
		m.BondingState, m.TokenPrice, m.MarketCap, m.Snapshot,
	)
}

func deepCopyMemeToken(src MemeToken) MemeToken {
	dst := src
	// Deep copy the Info slice
	dst.Info = make([]MemeInfo, len(src.Info))
	copy(dst.Info, src.Info)

	// Also deep copy BondingCurveState pointers, if you ever mutate them
	for i := range dst.Info {
		if src.Info[i].BondingState != nil {
			copyOfBC := *src.Info[i].BondingState // copy the struct by value
			dst.Info[i].BondingState = &copyOfBC
		}
	}
	return dst
}

func (ta TokenAnalysis) String() string {
	return fmt.Sprintf(
		"TokenAnalysis{MarketCapSufficiency: %t, ReservesSufficiency: %t, DataPoints: %d, ReserveRatio: %.4f, SolReserveRatio: %.4f, PriceStability: %t, PriceConvergence: %.10f, SimpleMovingAverage: %.10f, PercentageChange: %.3f%%, PriceTrendSlope: %.10f, ConsistentlyTrendingUp: %t, Volatility: %.10f, TheoreticalPrice: %.8f, CurrentPrice: %.8f, RiskRewardScore: %.8f, Time: %s}",
		ta.MarketCapSufficiency,
		ta.ReservesSufficiency,
		ta.DataPoints,
		ta.ReserveRatio,
		ta.SolReserveRatio,
		ta.PriceStability,
		ta.PriceConvergence,
		ta.SimpleMovingAverage,
		ta.PercentageChange,
		ta.PriceTrendSlope,
		ta.ConsistentlyTrendingUp,
		ta.Volatility,
		ta.TheoreticalPrice,
		ta.CurrentPrice,
		ta.RiskRewardScore,
		ta.Snapshot,
	)
}
