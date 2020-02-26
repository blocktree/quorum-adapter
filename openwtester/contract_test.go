/*
 * Copyright 2019 The openwallet Authors
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

package openwtester

import (
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
	"testing"
)

func TestCallSmartContractABI(t *testing.T) {
	tm := testInitWalletManager()
	walletID := "WDMxQukjqS4SAP73Hx9kkPT4eVgTXDjC3r"
	accountID := "CBGfADJdeDDPKh7wywxDrmkTJzxGjAQJyT4hVD44bvLE"

	contract := openwallet.SmartContract{
		Address:  "0xd6a4fe256d317f0bd19bc250be2fd9e30e0adc2d",
		Symbol:   "QUORUM",
		Name:     "FUQI",
		Token:    "FUQI",
		Decimals: 2,
	}
	contract.SetABI(`[{"inputs":[{"internalType":"contract KeyValueStorage","name":"storage_","type":"address"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"previousOwner","type":"address"},{"indexed":true,"internalType":"address","name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"constant":true,"inputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"name":"_auctionList","outputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"uint256","name":"price","type":"uint256"},{"internalType":"uint256","name":"dealPrice","type":"uint256"},{"internalType":"address","name":"buyer","type":"address"},{"internalType":"address","name":"seller","type":"address"},{"internalType":"uint8","name":"status","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"name":"_winPrizeList","outputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"bytes32","name":"productID","type":"bytes32"},{"internalType":"uint8","name":"status","type":"uint8"},{"internalType":"address","name":"winner","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getOwner","outputs":[{"internalType":"address","name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"internalType":"address","name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"bytes32","name":"productID","type":"bytes32"},{"internalType":"address","name":"winner","type":"address"}],"name":"addWinPrize","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"uint256","name":"price","type":"uint256"},{"internalType":"address","name":"seller","type":"address"}],"name":"auctionPrize","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"address","name":"buyer","type":"address"},{"internalType":"uint256","name":"dealPrice","type":"uint256"}],"name":"dealAuction","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"}],"name":"receivePrize","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"internalType":"bytes32","name":"num","type":"bytes32"}],"name":"getWinPrizeInfo","outputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"bytes32","name":"productID","type":"bytes32"},{"internalType":"uint8","name":"status","type":"uint8"},{"internalType":"address","name":"winner","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"bytes32","name":"num","type":"bytes32"}],"name":"getAuctionInfo","outputs":[{"internalType":"bytes32","name":"number","type":"bytes32"},{"internalType":"uint256","name":"price","type":"uint256"},{"internalType":"uint256","name":"dealPrice","type":"uint256"},{"internalType":"address","name":"buyer","type":"address"},{"internalType":"address","name":"seller","type":"address"},{"internalType":"uint8","name":"status","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"}]`)

	callParam := []string{"getWinPrizeInfo", "0x9b588888ecf84e6d18758017ae4260361eb6177586a22a279bbd7c862dd01087"}

	result, err := tm.CallSmartContractABI(testApp, walletID, accountID, &contract, callParam)
	if err != nil {
		t.Errorf("CallSmartContractABI failed, unexpected error: %v", err)
		return
	}
	log.Infof("result: %+v", result)
}