package worker

import (
	"time"

	"github.com/anyswap/CrossChain-Router/v3/router/bridge"
	"github.com/anyswap/CrossChain-Router/v3/rpc/client"
)

const interval = 10 * time.Millisecond

// StartRouterSwapWork start router swap job
func StartRouterSwapWork(isServer bool) {
	logWorker("worker", "start router swap worker")

	client.InitHTTPClient()
	bridge.InitRouterBridges(isServer)

	go bridge.StartAdjustGatewayOrderJob()
	time.Sleep(interval)

	if !isServer {
		StartAcceptSignJob()
		return
	}

	StartSwapJob()
	time.Sleep(interval)

	go StartVerifyJob()
	time.Sleep(interval)

	go StartStableJob()
	time.Sleep(interval)

	go StartReplaceJob()
}
