/*
 * Copyright 2022 The openwallet Authors
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
	"testing"

	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openwallet"
)

func TestWalletManager_erc721_GetNFTBalanceByAddress(t *testing.T) {
	wm := testNewWalletManager()

	nft := &openwallet.NFT{
		Symbol:   "ETH",
		Address:  "0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D",
		Token:    "BoredApe",
		Name:     "BoredApeYachtClub",
		Protocol: openwallet.InterfaceTypeERC721,
		TokenID:  "5493",
	}
	owner := "0xe275c5f1714cc65ac667fb1be124aebd2d1ea5f9"

	balance, err := wm.NFTContractDecoder.GetNFTBalanceByAddress(nft, owner)

	if err != nil {
		t.Errorf("erc721_GetNFTBalanceByAddress error: %v", err)
		return
	}
	log.Infof("balance: %v", balance.Balance)
}

func TestWalletManager_erc721_GetNFTOwnerByTokenID(t *testing.T) {
	wm := testNewWalletManager()

	nft := &openwallet.NFT{
		Symbol:   "ETH",
		Address:  "0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D",
		Token:    "BoredApe",
		Name:     "BoredApeYachtClub",
		Protocol: openwallet.InterfaceTypeERC721,
		TokenID:  "5493",
	}
	owner, err := wm.NFTContractDecoder.GetNFTOwnerByTokenID(nft)
	if err != nil {
		t.Errorf("erc721_GetNFTOwnerByTokenID error: %v", err)
		return
	}
	log.Infof("owner: %v", owner.Owner)
}

func TestWalletManager_erc721_GetMetaDataOfNFT(t *testing.T) {
	wm := testNewWalletManager()

	nft := &openwallet.NFT{
		Symbol:   "ETH",
		Address:  "0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D",
		Token:    "BoredApe",
		Name:     "BoredApeYachtClub",
		Protocol: openwallet.InterfaceTypeERC721,
		TokenID:  "5493",
	}
	metaData, err := wm.NFTContractDecoder.GetMetaDataOfNFT(nft)
	if err != nil {
		t.Errorf("erc721_GetMetaDataOfNFT error: %v", err)
		return
	}
	log.Infof("metaData: %v", metaData.URI)
}

//func TestWalletManager_SupportsInterface(t *testing.T) {
//	wm := testNewWalletManager()
//
//	nft := &openwallet.NFT{
//		Symbol:   "ETH",
//		Address:  "0x76BE3b62873462d2142405439777e971754E8E77",
//		Token:    "BoredApe",
//		Name:     "BoredApeYachtClub",
//		Protocol: openwallet.InterfaceTypeERC721,
//		TokenID:  "5493",
//	}
//	interfaceType := wm.NFTContractDecoder.SupportsInterface(nft.Address)
//	log.Infof("interfaceType: %v", interfaceType)
//}

func TestWalletManager_erc721_GetNFTTransfer(t *testing.T) {
	wm := testNewWalletManager()
	event := &openwallet.SmartContractEvent{
		Contract: &openwallet.SmartContract{
			ContractID: "U9C3X+BEcs9MjWe2bQG78W0e5SoRv/I8o+jwKK49+9s=",
			Symbol:     "QUORUM",
			Address:    "0xe24c9e84115819af35a1f3142932996e0216cd44",
			Token:      "",
			Protocol:   "",
			Name:       "",
			Decimals:   2},
		Event: "Transfer",
		Value: `{"from":"0xc1bf88d8ac5f63d0911400530cf1ff96090c3589","to":"0xe95a1e8c39f70e94f7f6e3408429a22bb0b19241","tokenId":1414}`,
	}
	tx, err := wm.NFTContractDecoder.GetNFTTransfer(event)
	if err != nil {
		t.Errorf("GetNFTTransfer failed, err: %v", err)
	}
	log.Infof("tx: %v", tx)
}

func TestWalletManager_erc1155_GetNFTBalanceByAddress(t *testing.T) {
	wm := testNewWalletManager()

	nft := &openwallet.NFT{
		Symbol:   "ETH",
		Address:  "0x5BABc381C7E9EdCF02654a9C30d384dFE54dd4A1",
		Token:    "ERC1155",
		Name:     "ERC1155",
		Protocol: openwallet.InterfaceTypeERC1155,
		TokenID:  "17",
	}
	owner := "0xd6b8ec0775abdce1e385c763b71eacff3991bad7"

	balance, err := wm.NFTContractDecoder.GetNFTBalanceByAddress(nft, owner)

	if err != nil {
		t.Errorf("erc1155_GetNFTBalanceByAddress error: %v", err)
		return
	}
	log.Infof("balance: %v", balance.Balance)
}

func TestWalletManager_erc1155_GetMetaDataOfNFT(t *testing.T) {
	wm := testNewWalletManager()

	nft := &openwallet.NFT{
		Symbol:   "ETH",
		Address:  "0x5BABc381C7E9EdCF02654a9C30d384dFE54dd4A1",
		Token:    "ERC1155",
		Name:     "ERC1155",
		Protocol: openwallet.InterfaceTypeERC1155,
		TokenID:  "17",
	}
	metaData, err := wm.NFTContractDecoder.GetMetaDataOfNFT(nft)
	if err != nil {
		t.Errorf("erc1155_GetMetaDataOfNFT error: %v", err)
		return
	}
	log.Infof("metaData: %v", metaData.URI)
}

func TestWalletManager_erc1155_GetNFTTransfer(t *testing.T) {
	wm := testNewWalletManager()
	event := &openwallet.SmartContractEvent{
		Contract: &openwallet.SmartContract{
			ContractID: "U9C3X+BEcs9MjWe2bQG78W0e5SoRv/I8o+jwKK49+9s=",
			Symbol:     "QUORUM",
			Address:    "0x5babc381c7e9edcf02654a9c30d384dfe54dd4a1",
			Token:      "",
			Protocol:   "",
			Name:       "",
			Decimals:   2},
		Event: "TransferSingle",
		Value: `{"from":"0x0000000000000000000000000000000000000000","id":17,"operator":"0x4e32004d8b81847a670b4a1778ace4dcf2bba01e","to":"0xd6b8ec0775abdce1e385c763b71eacff3991bad7","value":3}`,
	}
	tx, err := wm.NFTContractDecoder.GetNFTTransfer(event)
	if err != nil {
		t.Errorf("GetNFTTransfer failed, err: %v", err)
	}
	log.Infof("tx: %v", tx)
}
