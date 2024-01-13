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
	txid := "0xf4783b61c9e3ceb33598b4d67c9e8f7ca3c5d6b20c21ba8e0378b512e91c7208"
	tx, err := wm.GetTransactionByHash(txid)
	if err != nil {
		t.Errorf("get transaction by has failed, err=%v", err)
		return
	}
	log.Infof("tx: %+v", tx)
}

func TestWalletManager_ethGetTransactionReceipt(t *testing.T) {
	wm := testNewWalletManager()
	//0x4e9d76f0fce70c5a1f376983bf710016a6344e0bc026f8795b8a03a71d85dd0e
	//0x3cf48e3af2df9149725c909f5e4553c9565c760e8094628b982e545373d1a660
	txid := "0x34cec138ac784154ee6093a1bcfd8f7f66cb6f2760d8abeeda25bef0fcb474c7"
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
		"0x58b332acc6f24ce1adf75bf32e66852df5cea89a": openwallet.ScanTargetResult{
			SourceKey: "receiver",
			Exist:     true,
		},

		"0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174": openwallet.ScanTargetResult{
			SourceKey: "GOOD",
			Exist:     true,
			TargetInfo: &openwallet.SmartContract{
				ContractID: "GOOD",
				Symbol:     "QUORUM",
				Address:    "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174",
				Token:      "FUQI",
				Protocol:   "",
				Name:       "FUQI",
				Decimals:   2,
			},
		},
	}
	txid := "0x4671340d0a22ace1f3ebff5831b037aabab96a9090e3499e5ba2dee79de618fc"
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

func TestBlockScanner_GetBlockchainSyncStatus(t *testing.T) {
	wm := testNewWalletManager()
	status, err := wm.GetBlockScanner().GetBlockchainSyncStatus()
	if err != nil {
		t.Errorf("GetBlockchainSyncStatus failed, err=%v", err)
		return
	}
	log.Infof("status: %+v", status)
}
