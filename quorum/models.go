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
	"encoding/json"
	"fmt"
	"github.com/blocktree/openwallet/v2/common"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcom "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/v2/hdkeystore"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openwallet"
)

const (
	ERC20_ABI_JSON = `[{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[],"payable":false,"type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
)

var (
	ERC20_ABI, _ = abi.JSON(strings.NewReader(ERC20_ABI_JSON))
)

type EthBlock struct {
	BlockHeader
	Transactions []*BlockTransaction `json:"transactions"`
}

func (block *EthBlock) CreateOpenWalletBlockHeader() *openwallet.BlockHeader {
	header := &openwallet.BlockHeader{
		Hash:              block.BlockHash,
		Previousblockhash: block.PreviousHash,
		Height:            block.BlockHeight,
		Time:              uint64(time.Now().Unix()),
	}
	return header
}

type ERC20Token struct {
	Address  string `json:"address" storm:"id"`
	Symbol   string `json:"symbol" storm:"index"`
	Name     string `json:"name"`
	Decimals int    `json:"decimals"`
	balance  *big.Int
}

type EthEvent struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `josn:"data"`
	//BlockNumber string
	LogIndex string `json:"logIndex"`
	Removed  bool   `json:"removed"`
}

type TransactionReceipt struct {
	ETHReceipt *types.Receipt
	Raw        string
}

type TransferEvent struct {
	ContractAddress string
	TokenFrom       string
	TokenTo         string
	From            ethcom.Address
	To              ethcom.Address
	Value           *big.Int
}

func (receipt *TransactionReceipt) ParseTransferEvent() map[string][]*TransferEvent {
	var (
		transferEvents = make(map[string][]*TransferEvent)
		err            error
	)

	bc := bind.NewBoundContract(ethcom.HexToAddress("0x0"), ERC20_ABI, nil, nil, nil)
	for _, log := range receipt.ETHReceipt.Logs {

		event, _ := ERC20_ABI.EventByID(log.Topics[0])
		if event == nil || event.Name != "Transfer" {
			continue
		}

		var transfer TransferEvent
		err = bc.UnpackLog(&transfer, "Transfer", *log)
		if err != nil {
			continue
		}
		address := strings.ToLower(log.Address.String())
		events := transferEvents[address]
		if events == nil {
			events = make([]*TransferEvent, 0)
		}
		transfer.ContractAddress = address
		transfer.TokenFrom = strings.ToLower(transfer.From.String())
		transfer.TokenTo = strings.ToLower(transfer.To.String())
		events = append(events, &transfer)
		transferEvents[address] = events
	}

	return transferEvents
}

type Address struct {
	Address      string `json:"address" storm:"id"`
	Account      string `json:"account" storm:"index"`
	HDPath       string `json:"hdpath"`
	Index        int
	PublicKey    string
	balance      *big.Int //string `json:"balance"`
	tokenBalance *big.Int
	TxCount      uint64
	CreatedAt    time.Time
}

func (this *Address) CalcPrivKey(masterKey *hdkeystore.HDKey) ([]byte, error) {
	childKey, _ := masterKey.DerivedKeyWithPath(this.HDPath, owcrypt.ECC_CURVE_SECP256K1)
	keyBytes, err := childKey.GetPrivateKeyBytes()
	if err != nil {
		log.Error("get private key bytes, err=", err)
		return nil, err
	}
	return keyBytes, nil
}

func (this *Address) CalcHexPrivKey(masterKey *hdkeystore.HDKey) (string, error) {
	prikey, err := this.CalcPrivKey(masterKey)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(prikey), nil
}

type BlockTransaction struct {
	Hash             string `json:"hash" storm:"id"`
	BlockNumber      string `json:"blockNumber" storm:"index"`
	BlockHash        string `json:"blockHash" storm:"index"`
	From             string `json:"from"`
	To               string `json:"to"`
	Gas              string `json:"gas"`
	GasPrice         string `json:"gasPrice"`
	Value            string `json:"value"`
	Data             string `json:"input"`
	TransactionIndex string `json:"transactionIndex"`
	Timestamp        string `json:"timestamp"`
	BlockHeight      uint64 //transaction scanning 的时候对其进行赋值
	FilterFunc       openwallet.BlockScanTargetFuncV2
	Status           uint64
	receipt          *TransactionReceipt
	decimal          int32
}

func (this *BlockTransaction) GetAmountEthString() string {
	amount, _ := hexutil.DecodeBig(this.Value)
	amountVal := common.BigIntToDecimals(amount, this.decimal)
	return amountVal.String()
}

func (this *BlockTransaction) GetTxFeeEthString() string {
	gasPrice, _ := hexutil.DecodeBig(this.GasPrice)
	gas := common.StringNumToBigIntWithExp(this.Gas, 0)
	fee := big.NewInt(0)
	fee.Mul(gasPrice, gas)
	feeprice := common.BigIntToDecimals(fee, this.decimal)
	return feeprice.String()
}

type BlockHeader struct {
	BlockNumber     string `json:"number" storm:"id"`
	BlockHash       string `json:"hash"`
	GasLimit        string `json:"gasLimit"`
	GasUsed         string `json:"gasUsed"`
	Miner           string `json:"miner"`
	Difficulty      string `json:"difficulty"`
	TotalDifficulty string `json:"totalDifficulty"`
	PreviousHash    string `json:"parentHash"`
	BlockHeight     uint64 //RecoverBlockHeader的时候进行初始化
}

type txFeeInfo struct {
	GasLimit *big.Int
	GasPrice *big.Int
	Fee      *big.Int
}

func (txFee *txFeeInfo) CalcFee() error {
	fee := new(big.Int)
	fee.Mul(txFee.GasLimit, txFee.GasPrice)
	txFee.Fee = fee
	return nil
}

type CallMsg struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Data  string `json:"data"`
	Value string `json:"value"`
}

type CallResult map[string]interface{}

func (r CallResult) MarshalJSON() ([]byte, error) {
	newR := make(map[string]interface{})
	for key, value := range r {
		val := reflect.ValueOf(value) //读取变量的值，可能是指针或值
		if isByteArray(val.Type()) {
			newR[key] = toHex(value)
		} else {
			newR[key] = value
		}
	}
	return json.Marshal(newR)
}

func toHex(key interface{}) string {
	return fmt.Sprintf("0x%x", key)
}

func isByteArray(typ reflect.Type) bool {
	return (typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array) && isByte(typ.Elem())
}

func isByte(typ reflect.Type) bool {
	return typ.Kind() == reflect.Uint8
}
