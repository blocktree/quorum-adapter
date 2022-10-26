package openwtester

import (
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openw"
	"github.com/blocktree/quorum-adapter/quorum"
)

const (
	ChainSymbol = "QUORUM"
)

func init() {
	//注册钱包管理工具
	log.Notice("Wallet Manager Load Successfully.")
	openw.RegAssets(ChainSymbol, quorum.NewWalletManagerWithSymbol(ChainSymbol))
}
