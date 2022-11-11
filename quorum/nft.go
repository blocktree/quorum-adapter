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
	"github.com/tidwall/gjson"
	"math/big"
	"strings"

	"github.com/blocktree/openwallet/v2/openwallet"
	ethcom "github.com/ethereum/go-ethereum/common"
)

type NFTContractDecoder struct {
	*openwallet.NFTContractDecoderBase
	wm *WalletManager
}

// GetNFTBalanceByAddress 查询地址NFT余额列表
// NFT.TokenID为空则查询合约下拥有者所NFT数量。
func (decoder *NFTContractDecoder) GetNFTBalanceByAddress(nft *openwallet.NFT, owner string) (*openwallet.NFTBalance, *openwallet.Error) {

	balance := big.NewInt(0)
	switch nft.Protocol {
	case openwallet.InterfaceTypeERC721:
		if len(nft.TokenID) > 0 {
			//查询tokenID是否属于owner
			nftOwner, _ := decoder.GetNFTOwnerByTokenID(nft)
			if nftOwner != nil && nftOwner.Owner == strings.ToLower(owner) {
				balance = big.NewInt(1)
			} else {
				balance = big.NewInt(0)
			}
		} else {
			//call `balanceOf(address)`
			result, err := decoder.wm.CallABI(nft.Address, ERC721_ABI, "balanceOf", owner)
			if err != nil {
				return nil, err
			}
			balanceRes, ok := result["balance"].(*big.Int)
			if ok {
				balance = balanceRes
			}
		}

	case openwallet.InterfaceTypeERC1155:
		result, err := decoder.wm.CallABI(nft.Address, ERC1155_ABI, "balanceOf", owner, nft.TokenID)
		if err != nil {
			return nil, err
		}
		balanceRes, ok := result[""].(*big.Int)
		if ok {
			balance = balanceRes
		}
	default:
		return nil, openwallet.Errorf(openwallet.ErrSystemException, "NFT interface type is invalid")
	}

	nftBalance := &openwallet.NFTBalance{
		NFT:     nft,
		Balance: balance.String(),
	}

	return nftBalance, nil
}

// GetNFTBalanceByAddressBatch 查询地址NFT余额列表
func (decoder *NFTContractDecoder) GetNFTBalanceByAddressBatch(nft []*openwallet.NFT, owner []string) ([]*openwallet.NFTBalance, *openwallet.Error) {
	if len(nft) != len(owner) {
		return nil, openwallet.Errorf(openwallet.ErrSystemException, "NFT array length is not equal to owner array length")
	}
	contractAddress := ""
	ids := ""
	owners := ""
	for i, o := range nft {

		if len(o.TokenID) == 0 {
			return nil, openwallet.Errorf(openwallet.ErrSystemException, "NFT token id is empty")
		}

		if i == 0 {
			contractAddress = o.Address
			ids = o.Address
			owners = owner[i]
		} else {
			ids = ids + "," + o.Address
			owners = owners + "," + owner[i]
		}
		if contractAddress != o.Address {
			return nil, openwallet.Errorf(openwallet.ErrSystemException, "NFTs contract address is inconsistent")
		}

	}
	result, err := decoder.wm.CallABI(contractAddress, ERC1155_ABI, "balanceOfBatch", owners, ids)
	if err != nil {
		return nil, err
	}
	balanceRes, ok := result[""].([]*big.Int)
	balances := make([]*openwallet.NFTBalance, 0)
	if ok {
		for i, re := range balanceRes {
			nftBalance := &openwallet.NFTBalance{
				NFT:     nft[i],
				Balance: re.String(),
			}
			balances = append(balances, nftBalance)
		}
	}
	return balances, nil
}

// GetNFTOwnerByTokenID 查询地址token的拥有者
func (decoder *NFTContractDecoder) GetNFTOwnerByTokenID(nft *openwallet.NFT) (*openwallet.NFTOwner, *openwallet.Error) {

	if len(nft.TokenID) == 0 {
		return nil, openwallet.Errorf(openwallet.ErrSystemException, "NFT token id is empty")
	}

	owner := ""
	switch nft.Protocol {
	case openwallet.InterfaceTypeERC721:
		result, err := decoder.wm.CallABI(nft.Address, ERC721_ABI, "ownerOf", nft.TokenID)
		if err != nil {
			return nil, err
		}
		ownerRes, ok := result["owner"].(ethcom.Address)
		if ok {
			owner = strings.ToLower(ownerRes.String())
		}
	default:
		return nil, openwallet.Errorf(openwallet.ErrSystemException, "NFT interface type is not support")
	}

	nftOwner := &openwallet.NFTOwner{
		NFT:   nft,
		Owner: owner,
	}

	return nftOwner, nil

}

// GetMetaDataOfNFT 查询NFT的MetaData
func (decoder *NFTContractDecoder) GetMetaDataOfNFT(nft *openwallet.NFT) (*openwallet.NFTMetaData, *openwallet.Error) {

	if len(nft.TokenID) == 0 {
		return nil, openwallet.Errorf(openwallet.ErrSystemException, "NFT token id is empty")
	}
	uri := ""
	switch nft.Protocol {
	case openwallet.InterfaceTypeERC721:
		result, err := decoder.wm.CallABI(nft.Address, ERC721_ABI, "tokenURI", nft.TokenID)
		if err != nil {
			return nil, err
		}
		uriRes, ok := result[""].(string)
		if ok {
			uri = uriRes
		}
	case openwallet.InterfaceTypeERC1155:
		result, err := decoder.wm.CallABI(nft.Address, ERC1155_ABI, "uri", nft.TokenID)
		if err != nil {
			return nil, err
		}
		uriRes, ok := result[""].(string)
		if ok {
			uri = uriRes
		}
	default:
		return nil, openwallet.Errorf(openwallet.ErrSystemException, "NFT interface type is not support")
	}

	nftMetaData := &openwallet.NFTMetaData{
		NFT: nft,
		URI: uri,
	}

	return nftMetaData, nil

}

// GetNFTTransfer 从event解析NFT转账信息
func (decoder *NFTContractDecoder) GetNFTTransfer(event *openwallet.SmartContractEvent) (*openwallet.NFTTransfer, *openwallet.Error) {
	if event == nil {
		return nil, openwallet.Errorf(openwallet.ErrSystemException, "SmartContractEvent is nil")
	}
	var (
		nftTx     *openwallet.NFTTransfer
		from      = ""
		to        = ""
		operator  = ""
		nfts      = make([]openwallet.NFT, 0)
		amounts   = make([]string, 0)
		eventType = openwallet.NFTEventTypeTransferred
	)
	// 检查合约是否支持nft协议
	inferfaceType := decoder.wm.SupportsInterface(event.Contract.Address)
	obj := gjson.ParseBytes([]byte(event.Value))
	switch inferfaceType {
	case openwallet.InterfaceTypeERC721:
		if event.Event == "Transfer" {
			//{"from":"0x1234","to":"0xabcd","tokenId":1414}}
			operator = obj.Get("from").String()
			from = obj.Get("from").String()
			to = obj.Get("to").String()
			amounts = append(amounts, "1")
			nfts = append(nfts, openwallet.NFT{
				Symbol:   event.Contract.Symbol,
				Address:  event.Contract.Address,
				Token:    event.Contract.Token,
				Protocol: inferfaceType,
				Name:     event.Contract.Name,
				TokenID:  obj.Get("tokenId").String(),
			})
		} else {
			return nil, openwallet.Errorf(openwallet.ErrSystemException, "NFT event invalid")
		}
	case openwallet.InterfaceTypeERC1155:

		operator = obj.Get("operator").String()
		from = obj.Get("from").String()
		to = obj.Get("to").String()

		if event.Event == "TransferSingle" {
			nfts = append(nfts, openwallet.NFT{
				Symbol:   event.Contract.Symbol,
				Address:  event.Contract.Address,
				Token:    event.Contract.Token,
				Protocol: inferfaceType,
				Name:     event.Contract.Name,
				TokenID:  obj.Get("id").String(),
			})
			amounts = append(amounts, obj.Get("value").String())
		} else if event.Event == "TransferBatch" {
			ids := obj.Get("ids").Array()
			values := obj.Get("values").Array()
			for i, id := range ids {
				nfts = append(nfts, openwallet.NFT{
					Symbol:   event.Contract.Symbol,
					Address:  event.Contract.Address,
					Token:    event.Contract.Token,
					Protocol: inferfaceType,
					Name:     event.Contract.Name,
					TokenID:  id.String(),
				})
				amounts = append(amounts, values[i].String())
			}
		} else {
			return nil, openwallet.Errorf(openwallet.ErrSystemException, "NFT event invalid")
		}

	default:
		return nil, openwallet.Errorf(openwallet.ErrSystemException, "NFT interface type is not support")
	}

	if from == ethcom.HexToAddress("0x00").String() {
		eventType = openwallet.NFTEventTypeMinted
	}
	if to == ethcom.HexToAddress("0x00").String() {
		eventType = openwallet.NFTEventTypeBurned
	}

	nftTx = &openwallet.NFTTransfer{
		Tokens:    nfts,
		Operator:  operator,
		From:      from,
		To:        to,
		Amounts:   amounts,
		EventType: uint64(eventType),
	}

	return nftTx, nil
}
