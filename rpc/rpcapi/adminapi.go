package rpcapi

import (
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/anyswap/CrossChain-Router/v3/admin"
	"github.com/anyswap/CrossChain-Router/v3/common"
	"github.com/anyswap/CrossChain-Router/v3/log"
	"github.com/anyswap/CrossChain-Router/v3/mongodb"
	"github.com/anyswap/CrossChain-Router/v3/params"
	"github.com/anyswap/CrossChain-Router/v3/router"
	"github.com/anyswap/CrossChain-Router/v3/tokens"
	"github.com/anyswap/CrossChain-Router/v3/worker"
)

const (
	maintainCmd     = "maintain"
	passbigvalueCmd = "passbigvalue"
	reswapCmd       = "reswap"
	replaceswapCmd  = "replaceswap"

	successReuslt = "Success"
)

// AdminCall admin call
func (s *RouterSwapAPI) AdminCall(r *http.Request, rawTx, result *string) (err error) {
	if !params.HasRouterAdmin() {
		return fmt.Errorf("no admin is configed")
	}
	tx, err := admin.DecodeTransaction(*rawTx)
	if err != nil {
		return err
	}
	sender, args, err := admin.VerifyTransaction(tx)
	if err != nil {
		return err
	}
	senderAddress := sender.String()
	if !params.IsRouterAdmin(senderAddress) {
		switch args.Method {
		case maintainCmd, reswapCmd:
			return fmt.Errorf("sender %v is not admin", senderAddress)
		case passbigvalueCmd, replaceswapCmd:
			if !params.IsRouterAssistant(senderAddress) {
				return fmt.Errorf("sender %v is not assistant", senderAddress)
			}
		default:
			return fmt.Errorf("unknown admin method '%v'", args.Method)
		}
	}
	log.Info("admin call", "caller", senderAddress, "args", args, "result", result)
	return doRouterAdminCall(args, result)
}

func doRouterAdminCall(args *admin.CallArgs, result *string) error {
	switch args.Method {
	case maintainCmd:
		return maintain(args, result)
	case passbigvalueCmd:
		return routerPassBigValue(args, result)
	case reswapCmd:
		return routerReswap(args, result)
	case replaceswapCmd:
		return routerReplaceSwap(args, result)
	default:
		return fmt.Errorf("unknown admin method '%v'", args.Method)
	}
}

func getKeys(args *admin.CallArgs, startPos int) (chainID, txid string, logIndex int, err error) {
	if len(args.Params) < startPos+3 {
		err = fmt.Errorf("wrong number of params, have %v want at least %v", len(args.Params), startPos+3)
		return
	}
	chainID = args.Params[startPos]
	if _, err = common.GetBigIntFromStr(chainID); err != nil || chainID == "" {
		err = fmt.Errorf("wrong chain id '%v'", chainID)
		return
	}
	txid = args.Params[startPos+1]
	if !common.IsHexHash(txid) {
		err = fmt.Errorf("wrong tx id '%v'", txid)
		return
	}
	logIndexStr := args.Params[startPos+2]
	logIndex, err = common.GetIntFromStr(logIndexStr)
	if err != nil {
		err = fmt.Errorf("wrong log index '%v'", logIndexStr)
	}
	return
}

func maintain(args *admin.CallArgs, result *string) (err error) {
	if len(args.Params) != 2 {
		return fmt.Errorf("wrong number of params, have %v want 2", len(args.Params))
	}
	action := args.Params[0]
	arguments := args.Params[1]

	actPause := "pause"
	actUnpause := "unpause"

	switch action {
	case actPause, actUnpause:
		chainIDs := strings.Split(arguments, ",")
		if action == actPause {
			router.AddPausedChainIDs(chainIDs)
		} else {
			router.RemovePausedChainIDs(chainIDs)
		}
		log.Infof("after action %v, the paused chainIDs are %v", action, router.GetPausedChainIDs())
	default:
		return fmt.Errorf("unkown maintain action '%v'", action)
	}
	return nil
}

func getGasPrice(args *admin.CallArgs, startPos int) (gasPrice *big.Int, err error) {
	if len(args.Params) < startPos+1 {
		err = fmt.Errorf("wrong number of params, have %v want at least %v", len(args.Params), startPos+3)
		return
	}
	gasPriceStr := args.Params[startPos]
	if gasPriceStr == "" {
		return
	}
	if gasPrice, err = common.GetBigIntFromStr(gasPriceStr); err != nil {
		err = fmt.Errorf("wrong gas price '%v'", gasPriceStr)
	}
	return
}

func routerPassBigValue(args *admin.CallArgs, result *string) (err error) {
	chainID, txid, logIndex, err := getKeys(args, 0)
	if err != nil {
		return err
	}
	bridge := router.GetBridgeByChainID(chainID)
	if bridge == nil {
		return tokens.ErrNoBridgeForChainID
	}
	verifyArgs := &tokens.VerifyArgs{
		SwapType:      tokens.ERC20SwapType,
		LogIndex:      logIndex,
		AllowUnstable: false,
	}
	swapInfo, err := bridge.VerifyTransaction(txid, verifyArgs)
	if err != nil {
		return err
	}
	err = mongodb.RouterAdminPassBigValue(chainID, txid, logIndex)
	if err != nil {
		return err
	}
	_ = worker.AddInitialSwapResult(swapInfo, mongodb.MatchTxEmpty)
	*result = successReuslt
	return nil
}

func routerReswap(args *admin.CallArgs, result *string) (err error) {
	chainID, txid, logIndex, err := getKeys(args, 0)
	if err != nil {
		return err
	}
	err = mongodb.RouterAdminReswap(chainID, txid, logIndex)
	if err != nil {
		return err
	}
	worker.DeleteCachedSwap(chainID, txid, logIndex)
	*result = successReuslt
	return nil
}

func routerReplaceSwap(args *admin.CallArgs, result *string) (err error) {
	chainID, txid, logIndex, err := getKeys(args, 0)
	if err != nil {
		return err
	}
	gasPrice, err := getGasPrice(args, 3)
	if err != nil {
		return err
	}
	res, err := mongodb.FindRouterSwapResult(chainID, txid, logIndex)
	if err != nil {
		return err
	}
	err = worker.ReplaceRouterSwap(res, gasPrice, true)
	if err != nil {
		return err
	}
	*result = successReuslt
	return nil
}
