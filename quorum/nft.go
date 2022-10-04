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

	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcom "github.com/ethereum/go-ethereum/common"
)

type NFTContractDecoder struct {
	*openwallet.NFTContractDecoderBase
	wm *WalletManager
}

func (decoder *NFTContractDecoder) CallABI(contractAddr string, abiInstance abi.ABI, abiParam ...string) (map[string]interface{}, *openwallet.Error) {

	methodName := ""
	if len(abiParam) > 0 {
		methodName = abiParam[0]
	}

	//abi编码
	data, err := decoder.wm.EncodeABIParam(abiInstance, abiParam...)
	if err != nil {
		return nil, openwallet.ConvertError(err)
	}

	callMsg := CallMsg{
		From:  ethcom.HexToAddress("0x00"),
		To:    ethcom.HexToAddress(contractAddr),
		Data:  data,
		Value: big.NewInt(0),
	}

	result, err := decoder.wm.EthCall(callMsg, "latest")
	if err != nil {
		return nil, openwallet.ConvertError(err)
	}

	rMap, _, err := decoder.wm.DecodeABIResult(abiInstance, methodName, result)
	if err != nil {
		return nil, openwallet.ConvertError(err)
	}

	return rMap, nil
}

func (decoder *NFTContractDecoder) SupportsInterface(contractAddr string) string {
	//is support erc721
	result721, err := decoder.CallABI(contractAddr, ERC721_ABI, "supportsInterface", "0x80ac58cd")
	if err != nil {
		log.Errorf("SupportsInterface: %+v", err)
	}
	//is support erc1155
	result1155, err := decoder.CallABI(contractAddr, ERC721_ABI, "supportsInterface", "0xd9b67a26")
	if err != nil {
		log.Errorf("SupportsInterface: %+v", err)
	}

	support721, ok := result721[""].(bool)
	if ok && support721 {
		return openwallet.InterfaceTypeERC721
	}

	support1155, ok := result1155[""].(bool)
	if ok && support1155 {
		return openwallet.InterfaceTypeERC1155
	}

	return openwallet.InterfaceTypeUnknown
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
			result, err := decoder.CallABI(nft.Address, ERC721_ABI, "balanceOf", owner)
			if err != nil {
				return nil, err
			}
			balanceRes, ok := result["balance"].(*big.Int)
			if ok {
				balance = balanceRes
			}
		}

	case openwallet.InterfaceTypeERC1155:
		balance = big.NewInt(0)
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
	return nil, openwallet.Errorf(openwallet.ErrSystemException, "GetNFTBalanceByAddress not implement")
}

//GetNFTOwnerByTokenID 查询地址token的拥有者
func (decoder *NFTContractDecoder) GetNFTOwnerByTokenID(nft *openwallet.NFT) (*openwallet.NFTOwner, *openwallet.Error) {

	owner := ""
	switch nft.Protocol {
	case openwallet.InterfaceTypeERC721:
		if len(nft.TokenID) > 0 {
			//call `ownerOf(tokenId)`
			result, err := decoder.CallABI(nft.Address, ERC721_ABI, "ownerOf", nft.TokenID)
			if err != nil {
				return nil, err
			}
			ownerRes, ok := result["owner"].(ethcom.Address)
			if ok {
				owner = strings.ToLower(ownerRes.String())
			}
		} else {
			return nil, openwallet.Errorf(openwallet.ErrSystemException, "NFT token id is empty")
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

//GetMetaDataOfNFT 查询NFT的MetaData
func (decoder *NFTContractDecoder) GetMetaDataOfNFT(nft *openwallet.NFT) (*openwallet.NFTMetaData, *openwallet.Error) {

	uri := ""
	switch nft.Protocol {
	case openwallet.InterfaceTypeERC721:
		if len(nft.TokenID) > 0 {
			//call `ownerOf(tokenId)`
			result, err := decoder.CallABI(nft.Address, ERC721_ABI, "tokenURI", nft.TokenID)
			if err != nil {
				return nil, err
			}
			uriRes, ok := result[""].(string)
			if ok {
				uri = uriRes
			}
		} else {
			return nil, openwallet.Errorf(openwallet.ErrSystemException, "NFT token id is empty")
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

//GetNFTTransfer 从event解析NFT转账信息
func (decoder *NFTContractDecoder) GetNFTTransfer(event *openwallet.SmartContractEvent) (*openwallet.NFTTransfer, *openwallet.Error) {
	if event == nil {
		return nil, openwallet.Errorf(openwallet.ErrSystemException, "SmartContractEvent is nil")
	}
	var (
		nftTx *openwallet.NFTTransfer
	)
	//TODO: 检查合约是否支持nft协议
	inferfaceType := decoder.SupportsInterface(event.Contract.Address)
	switch inferfaceType {
	case openwallet.InterfaceTypeERC721:
		//{"from":"0x1234","to":"0xabcd","tokenId":1414}}
		obj := gjson.ParseBytes([]byte(event.Value))
		nftTx = &openwallet.NFTTransfer{
			Tokens: []openwallet.NFT{
				openwallet.NFT{
					Symbol:   event.Contract.Symbol,
					Address:  event.Contract.Address,
					Token:    event.Contract.Token,
					Protocol: inferfaceType,
					Name:     event.Contract.Name,
					TokenID:  obj.Get("tokenId").String(),
				},
			},
			Operator:  obj.Get("from").String(),
			From:      obj.Get("from").String(),
			To:        obj.Get("to").String(),
			Amounts:   []string{"1"},
			EventType: 0,
		}
	default:
		return nil, openwallet.Errorf(openwallet.ErrSystemException, "NFT interface type is not support")
	}

	return nftTx, nil
}
