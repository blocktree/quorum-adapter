/*
 * Copyright 2018 The openwallet Authors
 * This file is part of the openwallet library.
 *
 * The openwallet library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The openwallet library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 */

package quorum

import (
	"encoding/hex"
	"fmt"
	"github.com/astaxie/beego/config"
	"github.com/blocktree/openwallet/v2/common"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcom "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"path/filepath"
	"strings"
	"testing"
)

const (
	//ChainSymbol = "QUORUM"
	//ChainSymbol = "MATIC"
	ChainSymbol = "ETH"
)

var (
	tw *WalletManager
)

func init() {

	tw = testNewWalletManager()
}

func testNewWalletManager() *WalletManager {
	wm := NewWalletManager()

	//读取配置
	absFile := filepath.Join("conf", fmt.Sprintf("%s.ini", ChainSymbol))
	//log.Debug("absFile:", absFile)
	c, err := config.NewConfig("ini", absFile)
	if err != nil {
		panic(err)
	}
	wm.LoadAssetsConfig(c)
	wm.WalletClient.Debug = true
	if wm.MoralisSDK != nil {
		wm.MoralisSDK.Debug = true
	}

	return wm
}

func TestFixGasLimit(t *testing.T) {
	fixGasLimitStr := "sfsd"
	fixGasLimit := new(big.Int)
	fixGasLimit.SetString(fixGasLimitStr, 10)
	fmt.Printf("fixGasLimit: %d\n", fixGasLimit.Int64())
}

func TestWalletManager_GetAddrBalance(t *testing.T) {
	wm := testNewWalletManager()
	balance, err := wm.GetAddrBalance("0x3440f720862aa7dfd4f86ecc78542b3ded900c02", "pending")
	if err != nil {
		t.Errorf("GetAddrBalance2 error: %v", err)
		return
	}
	ethB := common.BigIntToDecimals(balance, wm.Decimal())
	log.Infof("ethB: %v", ethB)
}

func TestWalletManager_SetNetworkChainID(t *testing.T) {
	wm := testNewWalletManager()
	id, err := wm.SetNetworkChainID()
	if err != nil {
		t.Errorf("SetNetworkChainID error: %v", err)
		return
	}
	log.Infof("chainID: %d", id)
}

func TestWalletManager_EncodeABIParam(t *testing.T) {
	wm := testNewWalletManager()
	abiJSON := `[{"inputs":[],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"owner","type":"address"},{"indexed":true,"internalType":"address","name":"spender","type":"address"},{"indexed":false,"internalType":"uint256","name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"from","type":"address"},{"indexed":true,"internalType":"address","name":"to","type":"address"},{"indexed":false,"internalType":"uint256","name":"value","type":"uint256"}],"name":"Transfer","type":"event"},{"constant":true,"inputs":[],"name":"DOMAIN_SEPARATOR","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"PERMIT_TYPEHASH","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"address","name":"","type":"address"},{"internalType":"address","name":"","type":"address"}],"name":"allowance","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"}],"name":"approve","outputs":[{"internalType":"bool","name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"name","outputs":[{"internalType":"string","name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"nonces","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"},{"internalType":"uint256","name":"deadline","type":"uint256"},{"internalType":"uint8","name":"v","type":"uint8"},{"internalType":"bytes32","name":"r","type":"bytes32"},{"internalType":"bytes32","name":"s","type":"bytes32"}],"name":"permit","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"internalType":"string","name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"}],"name":"transfer","outputs":[{"internalType":"bool","name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"}],"name":"transferFrom","outputs":[{"internalType":"bool","name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
	method := "transfer"

	to := "0x8C178b782fab1d0686D88bC16B31F80431098fa1"
	amount := "102300"

	abiInstance, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Errorf("abi.JSON error: %v", err)
		return
	}

	data, err := wm.EncodeABIParam(abiInstance, method, to, amount)
	if err != nil {
		t.Errorf("EncodeABIParam error: %v", err)
		return
	}
	log.Infof("data: %s", hex.EncodeToString(data))
}

func TestWalletManager_EthCall(t *testing.T) {
	wm := testNewWalletManager()
	abiJSON := `[{"inputs":[{"internalType":"contract KeyValueStorage","name":"storage_","type":"address"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"previousOwner","type":"address"},{"indexed":true,"internalType":"address","name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"constant":true,"inputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"name":"_auctionList","outputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"uint256","name":"price","type":"uint256"},{"internalType":"uint256","name":"dealPrice","type":"uint256"},{"internalType":"address","name":"buyer","type":"address"},{"internalType":"address","name":"seller","type":"address"},{"internalType":"uint8","name":"status","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"name":"_winPrizeList","outputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"bytes32","name":"productID","type":"bytes32"},{"internalType":"uint8","name":"status","type":"uint8"},{"internalType":"address","name":"winner","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getOwner","outputs":[{"internalType":"address","name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"internalType":"address","name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"bytes32","name":"productID","type":"bytes32"},{"internalType":"address","name":"winner","type":"address"}],"name":"addWinPrize","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"uint256","name":"price","type":"uint256"},{"internalType":"address","name":"seller","type":"address"}],"name":"auctionPrize","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"address","name":"buyer","type":"address"},{"internalType":"uint256","name":"dealPrice","type":"uint256"}],"name":"dealAuction","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"}],"name":"receivePrize","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"internalType":"bytes32","name":"num","type":"bytes32"}],"name":"getWinPrizeInfo","outputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"bytes32","name":"productID","type":"bytes32"},{"internalType":"uint8","name":"status","type":"uint8"},{"internalType":"address","name":"winner","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"bytes32","name":"num","type":"bytes32"}],"name":"getAuctionInfo","outputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"uint256","name":"price","type":"uint256"},{"internalType":"uint256","name":"dealPrice","type":"uint256"},{"internalType":"address","name":"buyer","type":"address"},{"internalType":"address","name":"seller","type":"address"},{"internalType":"uint8","name":"status","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"}]`
	method := "getWinPrizeInfo"
	from := "0x993fc86c887a6139b92531468da0f5e70bc86a34"
	contractAddress := "0x7d6478556e21AeEd74681B5110373ee9d1Fd0e49"
	number := "hello"

	abiInstance, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Errorf("abi.JSON error: %v", err)
		return
	}

	data, err := wm.EncodeABIParam(abiInstance, method, number)
	if err != nil {
		t.Errorf("EncodeABIParam error: %v", err)
		return
	}

	callMsg := CallMsg{
		From: ethcom.HexToAddress(from),
		To:   ethcom.HexToAddress(contractAddress),
		Data: data,
	}

	result, err := wm.EthCall(callMsg, "latest")
	if err != nil {
		t.Errorf("EthCall error: %v", err)
		return
	}

	log.Infof("result: %s", result)
	rMap, rJSON, err := wm.DecodeABIResult(abiInstance, method, result)
	if err != nil {
		t.Errorf("EthCall error: %v", err)
		return
	}
	log.Infof("rMap: %+v", rMap)
	log.Infof("rJSON: %s", rJSON)
}

func TestWalletManager_GetTransactionFeeEstimated(t *testing.T) {
	wm := testNewWalletManager()
	txFee, err := wm.GetTransactionFeeEstimated(
		"0x993fc86c887a6139b92531468da0f5e70bc86a34",
		"0x993fc86c887a6139b92531468da0f5e70bc86a34",
		big.NewInt(0),
		nil)
	if err != nil {
		t.Errorf("GetTransactionFeeEstimated error: %v", err)
		return
	}
	log.Infof("txfee: %v", txFee)
}

func TestWalletManager_GetTransactionCount(t *testing.T) {
	wm := testNewWalletManager()
	count, err := wm.GetTransactionCount("0x3440f720862aa7dfd4f86ecc78542b3ded900c02")
	if err != nil {
		t.Errorf("GetTransactionCount error: %v", err)
		return
	}
	log.Infof("count: %v", count)
}

func TestWalletManager_IsContract(t *testing.T) {
	wm := testNewWalletManager()
	a, err := wm.IsContract("0x3440f720862aa7dfd4f86ecc78542b3ded900c02")
	log.Infof("IsContract: %v", a)
	if err != nil {
		t.Errorf("IsContract error: %v", err)
		return
	}

	c, _ := wm.IsContract("0x627b11ead4eb39ebe61a70ab3d6fe145e5d06ab6")
	log.Infof("IsContract: %v", c)

}

func TestWalletManager_DecodeReceiptLogResult(t *testing.T) {
	wm := testNewWalletManager()
	//	abiJSON := `
	//[{"inputs":[{"internalType":"contract KeyValueStorage","name":"storage_","type":"address"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"previousOwner","type":"address"},{"indexed":true,"internalType":"address","name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"implementation","type":"address"}],"name":"Upgraded","type":"event"},{"payable":true,"stateMutability":"payable","type":"fallback"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getOwner","outputs":[{"internalType":"address","name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"implementation","outputs":[{"internalType":"address","name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"name","outputs":[{"internalType":"string","name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"internalType":"string","name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"internalType":"address","name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"internalType":"address","name":"impl","type":"address"}],"name":"upgradeTo","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
	abiJSON := ERC721_ABI_JSON
	logJSON := `
			{
                "logIndex": "0x0",
                "transactionIndex": "0x0",
                "transactionHash": "0x6a949727089705103e873c5dc9ebfaac79deb5fe5df0b9f02672988336130af9",
                "blockHash": "0xd80805f3b261f8dc9fd95a60030615c20ff1ca29ecb34101faf91512aedd9f2c",
                "blockNumber": "0x4b",
                "address": "0xf8afe0a06e27ddbd5ec8adbbd5cee5220c3d4d85",
                "data": "0x",
                "topics": [
                    "0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b",
                    "0x00000000000000000000000044f64ef4bc4952b133a9c4b07157770f048eebe9"
                ],
                "type": "mined"
            }
`
	var logObj types.Log
	err := logObj.UnmarshalJSON([]byte(logJSON))
	if err != nil {
		t.Errorf("UnmarshalJSON error: %v", err)
		return
	}

	abiInstance, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Errorf("abi.JSON error: %v", err)
		return
	}

	rMap, name, rJSON, err := wm.DecodeReceiptLogResult(abiInstance, logObj)
	if err != nil {
		t.Errorf("DecodeReceiptLogResult error: %v", err)
		return
	}
	log.Infof("rMap: %+v", rMap)
	log.Infof("name: %+v", name)
	log.Infof("rJSON: %s", rJSON)
}

func TestWalletManager_GetBlockWithReceipts(t *testing.T) {
	wm := testNewWalletManager()
	_, err := wm.GetQNBlockWithReceipts(51951403)
	if err != nil {
		t.Errorf("GetQNBlockWithReceipts error: %v", err)
		return
	}
	//log.Infof("result: %v", result.Raw)
}

func TestWalletManager_GetBlockByNum(t *testing.T) {
	wm := testNewWalletManager()
	_, err := wm.GetBlockByNum(19088084, true)
	if err != nil {
		t.Errorf("GetTransactionCount error: %v", err)
		return
	}
	//log.Infof("block header: %+v", block.BlockHeader)
	//for i, tx := range block.Transactions {
	//	log.Infof("tx[%d]: %+v", i, tx)
	//	//if tx.Receipt == nil {
	//	//	log.Infof("tx.Receipt[%d]: nil", i)
	//	//	continue
	//	//}
	//	//log.Infof("tx.Receipt[%d]: %+v", i, tx.Receipt)
	//}
}
