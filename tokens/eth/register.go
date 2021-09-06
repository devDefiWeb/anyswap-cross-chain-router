package eth

import (
	"errors"
	"strings"

	"github.com/anyswap/CrossChain-Router/v3/log"
	"github.com/anyswap/CrossChain-Router/v3/tokens"
)

// RegisterSwap api
func (b *Bridge) RegisterSwap(txHash string, args *tokens.RegisterArgs) ([]*tokens.SwapTxInfo, []error) {
	swapType := args.SwapType
	logIndex := args.LogIndex

	switch swapType {
	case tokens.ERC20SwapType:
		return b.registerRouterSwapTx(txHash, logIndex)
	case tokens.AnyCallSwapType:
		return b.registerAnyCallSwapTx(txHash, logIndex)
	default:
		return nil, []error{tokens.ErrSwapTypeNotSupported}
	}
}

// nolint:dupl // ok
func (b *Bridge) registerRouterSwapTx(txHash string, logIndex int) ([]*tokens.SwapTxInfo, []error) {
	commonInfo := &tokens.SwapTxInfo{SwapInfo: tokens.SwapInfo{RouterSwapInfo: &tokens.RouterSwapInfo{}}}
	commonInfo.SwapType = tokens.ERC20SwapType // SwapType
	commonInfo.Hash = strings.ToLower(txHash)  // Hash
	commonInfo.LogIndex = logIndex             // LogIndex

	receipt, err := b.getAndVerifySwapTxReceipt(commonInfo, true)
	if err != nil {
		return []*tokens.SwapTxInfo{commonInfo}, []error{err}
	}

	swapInfos := make([]*tokens.SwapTxInfo, 0)
	errs := make([]error, 0)
	startIndex, endIndex := 1, len(receipt.Logs)

	if logIndex != 0 {
		if logIndex >= endIndex || logIndex < 0 {
			return []*tokens.SwapTxInfo{commonInfo}, []error{tokens.ErrLogIndexOutOfRange}
		}
		startIndex = logIndex
		endIndex = logIndex + 1
	}

	for i := startIndex; i < endIndex; i++ {
		swapInfo := &tokens.SwapTxInfo{}
		*swapInfo = *commonInfo
		swapInfo.RouterSwapInfo = &tokens.RouterSwapInfo{}
		swapInfo.LogIndex = i // LogIndex
		err := b.verifyERC20SwapTxLog(swapInfo, receipt.Logs[i])
		switch {
		case errors.Is(err, tokens.ErrSwapoutLogNotFound),
			errors.Is(err, tokens.ErrTxWithWrongContract):
			continue
		case err == nil:
			err = b.checkRouterSwapInfo(swapInfo)
		default:
			log.Debug(b.ChainConfig.BlockChain+" register router swap error", "txHash", txHash, "logIndex", swapInfo.LogIndex, "err", err)
		}
		swapInfos = append(swapInfos, swapInfo)
		errs = append(errs, err)
	}

	if len(swapInfos) == 0 {
		return []*tokens.SwapTxInfo{commonInfo}, []error{tokens.ErrSwapoutLogNotFound}
	}

	return swapInfos, errs
}
