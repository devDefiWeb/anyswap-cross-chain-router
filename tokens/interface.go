package tokens

import (
	"math/big"
)

// IMPCSign interface
type IMPCSign interface {
	VerifyMsgHash(rawTx interface{}, msgHash []string) error
	MPCSignTransaction(rawTx interface{}, args *BuildTxArgs) (signedTx interface{}, txHash string, err error)
}

// IBridgeConfg interface
type IBridgeConfg interface {
	GetGatewayConfig() *GatewayConfig
	GetChainConfig() *ChainConfig
	GetTokenConfig(token string) *TokenConfig

	InitGatewayConfig(chainID *big.Int)
	InitChainConfig(chainID *big.Int)
	InitTokenConfig(tokenID string, chainID *big.Int)
}

// IBridge interface
type IBridge interface {
	IBridgeConfg
	IMPCSign

	RegisterSwap(txHash string, args *RegisterArgs) ([]*SwapTxInfo, []error)
	VerifyTransaction(txHash string, ars *VerifyArgs) (*SwapTxInfo, error)
	BuildRawTransaction(args *BuildTxArgs) (rawTx interface{}, err error)
	SendTransaction(signedTx interface{}) (txHash string, err error)

	GetTransaction(txHash string) (interface{}, error)
	GetTransactionStatus(txHash string) *TxStatus
	GetLatestBlockNumber() (uint64, error)

	GetBigValueThreshold(token string) *big.Int
	IsValidAddress(address string) bool
}

// NonceSetter interface (for eth-like)
type NonceSetter interface {
	GetPoolNonce(address, height string) (uint64, error)
	SetNonce(pairID string, value uint64)
	AdjustNonce(pairID string, value uint64) (nonce uint64)
	IncreaseNonce(pairID string, value uint64)
}
