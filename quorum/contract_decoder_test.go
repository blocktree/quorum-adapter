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

func TestWalletManager_GetTokenBalanceByAddress(t *testing.T) {
	wm := testNewWalletManager()

	contract := openwallet.SmartContract{
		Address:  "0x550cdb1020046b3115a4f8ccebddfb28b66beb27",
		Symbol:   "QUORUM",
		Name:     "FQ",
		Token:    "FQ",
		Decimals: 2,
	}

	tokenBalances, err := wm.ContractDecoder.GetTokenBalanceByAddress(contract, "0x76b932e7ef077eabebe8a5064b99120ec81299ca")
	if err != nil {
		t.Errorf("GetTokenBalanceByAddress unexpected error: %v", err)
		return
	}
	for _, b := range tokenBalances {
		t.Logf("token balance: %+v", b.Balance)
	}
}

func TestWalletManager_GetTokenMetadata(t *testing.T) {
	wm := testNewWalletManager()

	tokenData, err := wm.ContractDecoder.GetTokenMetadata("0x7ceb23fd6bc0add59e62ac25578270cff1b9f619")
	if err != nil {
		t.Errorf("GetTokenMetadata unexpected error: %v", err)
		return
	}
	log.Infof("token metadata: %+v", tokenData)
}
