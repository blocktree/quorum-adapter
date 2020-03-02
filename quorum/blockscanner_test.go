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
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openwallet"
	"testing"
)

func TestWalletManager_EthGetTransactionByHash(t *testing.T) {
	wm := testNewWalletManager()
	txid := "0x19aabbc69d5d7ce4be4da8091dcfc12e83c4972a4660897cf3d3619abc208006"
	tx, err := wm.GetTransactionByHash(txid)
	if err != nil {
		t.Errorf("get transaction by has failed, err=%v", err)
		return
	}
	log.Infof("tx: %+v", tx)
}

func TestWalletManager_ethGetTransactionReceipt(t *testing.T) {
	wm := testNewWalletManager()
	txid := "0x6a949727089705103e873c5dc9ebfaac79deb5fe5df0b9f02672988336130af9"
	tx, err := wm.GetTransactionReceipt(txid)
	if err != nil {
		t.Errorf("get transaction by has failed, err=%v", err)
		return
	}
	log.Infof("tx: %+v", tx)
}

func TestWalletManager_EthGetBlockNumber(t *testing.T) {
	wm := testNewWalletManager()
	maxBlockHeight, err := wm.GetBlockNumber()
	if err != nil {
		t.Errorf("EthGetBlockNumber failed, err=%v", err)
		return
	}
	log.Infof("maxBlockHeight: %v", maxBlockHeight)
}

func TestBlockScanner_ExtractTransactionAndReceiptData(t *testing.T) {
	wm := testNewWalletManager()

	addrs := map[string]openwallet.ScanTargetResult{
		"0x3440f720862aa7dfd4f86ecc78542b3ded900c02": openwallet.ScanTargetResult{
			SourceKey: "receiver",
			Exist:     true,
		},

		"0xbff77bb15fec867b7db7b18a34fca6d20712ce2b": openwallet.ScanTargetResult{
			SourceKey: "GOOD",
			Exist:     true,
			TargetInfo: &openwallet.SmartContract{
				ContractID: "GOOD",
				Symbol:     "QUORUM",
				Address:    "0xbff77bb15fec867b7db7b18a34fca6d20712ce2b",
				Token:      "FUQI",
				Protocol:   "",
				Name:       "FUQI",
				Decimals:   2,
			},
		},
	}
	txid := "0xda660592894dd357eedadbb69c82d7ff57859d6fb6269d2ea2eab0dce1dfd8e1"
	scanTargetFunc := func(target openwallet.ScanTargetParam) openwallet.ScanTargetResult {
		return addrs[target.ScanTarget]
	}
	result, contractResult, err := wm.GetBlockScanner().ExtractTransactionAndReceiptData(txid, scanTargetFunc)
	if err != nil {
		t.Errorf("ExtractTransactionAndReceiptData failed, err=%v", err)
		return
	}

	for sourceKey, keyData := range result {
		log.Notice("account:", sourceKey)
		for _, data := range keyData {

			for i, input := range data.TxInputs {
				log.Std.Notice("data.TxInputs[%d]: %+v", i, input)
			}

			for i, output := range data.TxOutputs {
				log.Std.Notice("data.TxOutputs[%d]: %+v", i, output)
			}

			log.Std.Notice("data.Transaction: %+v", data.Transaction)
		}
	}

	for sourceKey, keyData := range contractResult {
		log.Notice("contractID:", sourceKey)
		log.Std.Notice("data.ContractTransaction: %+v", keyData)
	}
}
