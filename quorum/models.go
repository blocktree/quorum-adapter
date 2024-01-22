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
	"github.com/shopspring/decimal"
	"math/big"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/blocktree/openwallet/v2/common"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcom "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/tidwall/gjson"

	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/v2/hdkeystore"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openwallet"
)

const (
	ERC20_ABI_JSON   = `[{"inputs":[],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"owner","type":"address"},{"indexed":true,"internalType":"address","name":"spender","type":"address"},{"indexed":false,"internalType":"uint256","name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"from","type":"address"},{"indexed":true,"internalType":"address","name":"to","type":"address"},{"indexed":false,"internalType":"uint256","name":"value","type":"uint256"}],"name":"Transfer","type":"event"},{"constant":true,"inputs":[],"name":"DOMAIN_SEPARATOR","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"PERMIT_TYPEHASH","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"address","name":"","type":"address"},{"internalType":"address","name":"","type":"address"}],"name":"allowance","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"}],"name":"approve","outputs":[{"internalType":"bool","name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"name","outputs":[{"internalType":"string","name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"nonces","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"},{"internalType":"uint256","name":"deadline","type":"uint256"},{"internalType":"uint8","name":"v","type":"uint8"},{"internalType":"bytes32","name":"r","type":"bytes32"},{"internalType":"bytes32","name":"s","type":"bytes32"}],"name":"permit","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"internalType":"string","name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"}],"name":"transfer","outputs":[{"internalType":"bool","name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"}],"name":"transferFrom","outputs":[{"internalType":"bool","name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
	ERC721_ABI_JSON  = `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"owner","type":"address"},{"indexed":true,"internalType":"address","name":"approved","type":"address"},{"indexed":true,"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"owner","type":"address"},{"indexed":true,"internalType":"address","name":"operator","type":"address"},{"indexed":false,"internalType":"bool","name":"approved","type":"bool"}],"name":"ApprovalForAll","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"from","type":"address"},{"indexed":true,"internalType":"address","name":"to","type":"address"},{"indexed":true,"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"Transfer","type":"event"},{"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"approve","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"balance","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"getApproved","outputs":[{"internalType":"address","name":"operator","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"operator","type":"address"}],"name":"isApprovedForAll","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"name","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"ownerOf","outputs":[{"internalType":"address","name":"owner","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"safeTransferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"},{"internalType":"bytes","name":"data","type":"bytes"}],"name":"safeTransferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"operator","type":"address"},{"internalType":"bool","name":"_approved","type":"bool"}],"name":"setApprovalForAll","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"bytes4","name":"interfaceId","type":"bytes4"}],"name":"supportsInterface","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"symbol","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"index","type":"uint256"}],"name":"tokenByIndex","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"owner","type":"address"},{"internalType":"uint256","name":"index","type":"uint256"}],"name":"tokenOfOwnerByIndex","outputs":[{"internalType":"uint256","name":"tokenId","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"tokenURI","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"totalSupply","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"transferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"string","name":"collectionId","type":"string"}],"name":"CollectionCreate","type":"event"}]`
	ERC1155_ABI_JSON = `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"account","type":"address"},{"indexed":true,"internalType":"address","name":"operator","type":"address"},{"indexed":false,"internalType":"bool","name":"approved","type":"bool"}],"name":"ApprovalForAll","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"operator","type":"address"},{"indexed":true,"internalType":"address","name":"from","type":"address"},{"indexed":true,"internalType":"address","name":"to","type":"address"},{"indexed":false,"internalType":"uint256[]","name":"ids","type":"uint256[]"},{"indexed":false,"internalType":"uint256[]","name":"values","type":"uint256[]"}],"name":"TransferBatch","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"operator","type":"address"},{"indexed":true,"internalType":"address","name":"from","type":"address"},{"indexed":true,"internalType":"address","name":"to","type":"address"},{"indexed":false,"internalType":"uint256","name":"id","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"value","type":"uint256"}],"name":"TransferSingle","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"string","name":"value","type":"string"},{"indexed":true,"internalType":"uint256","name":"id","type":"uint256"}],"name":"URI","type":"event"},{"inputs":[{"internalType":"address","name":"account","type":"address"},{"internalType":"uint256","name":"id","type":"uint256"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address[]","name":"accounts","type":"address[]"},{"internalType":"uint256[]","name":"ids","type":"uint256[]"}],"name":"balanceOfBatch","outputs":[{"internalType":"uint256[]","name":"","type":"uint256[]"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"account","type":"address"},{"internalType":"address","name":"operator","type":"address"}],"name":"isApprovedForAll","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256[]","name":"ids","type":"uint256[]"},{"internalType":"uint256[]","name":"amounts","type":"uint256[]"},{"internalType":"bytes","name":"data","type":"bytes"}],"name":"safeBatchTransferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"id","type":"uint256"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"bytes","name":"data","type":"bytes"}],"name":"safeTransferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"operator","type":"address"},{"internalType":"bool","name":"approved","type":"bool"}],"name":"setApprovalForAll","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"bytes4","name":"interfaceId","type":"bytes4"}],"name":"supportsInterface","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"id","type":"uint256"}],"name":"uri","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"name","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"symbol","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"string","name":"collectionId","type":"string"}],"name":"CollectionCreate","type":"event"}]`
)

var (
	ERC20_ABI, _   = abi.JSON(strings.NewReader(ERC20_ABI_JSON))
	ERC721_ABI, _  = abi.JSON(strings.NewReader(ERC721_ABI_JSON))
	ERC1155_ABI, _ = abi.JSON(strings.NewReader(ERC1155_ABI_JSON))
)

type EthBlock struct {
	BlockHeader
	Transactions []*BlockTransaction `json:"transactions"`
}

func parseBlockFormMoralis(jsonData *gjson.Result, decimals int32) *EthBlock {

	//解析区块头
	block := &EthBlock{}
	block.BlockNumber = hexutil.EncodeUint64(jsonData.Get("number").Uint())
	block.BlockHash = jsonData.Get("hash").String()
	block.PreviousHash = jsonData.Get("parent_hash").String()
	block.BlockHeight = jsonData.Get("number").Uint()

	//解析交易列表
	blockTxs := make([]*BlockTransaction, 0)
	txs := jsonData.Get("transactions").Array()
	if txs != nil && len(txs) > 0 {
		for _, tx := range txs {
			blockTx := &BlockTransaction{}
			blockTx.Hash = tx.Get("hash").String()
			blockTx.BlockNumber = block.BlockNumber
			blockTx.BlockHash = block.BlockHash
			blockTx.From = tx.Get("from_address").String()
			blockTx.To = tx.Get("to_address").String()
			blockTx.Gas = tx.Get("receipt_gas_used").String()
			blockTx.GasPrice = hexutil.EncodeBig(common.StringNumToBigIntWithExp(tx.Get("gas_price").String(), 0))
			blockTx.Value = hexutil.EncodeBig(common.StringNumToBigIntWithExp(tx.Get("value").String(), 0))
			blockTx.BlockHeight = block.BlockHeight
			blockTx.Status = tx.Get("receipt_status").Uint()
			//blockTx.IsInternal = false
			blockTx.Decimal = decimals

			blockTxReceipt := &TransactionReceipt{ETHReceipt: &types.Receipt{}}
			blockTxReceipt.ETHReceipt.Status = blockTx.Status
			blockTxReceipt.ETHReceipt.TxHash = ethcom.HexToHash(blockTx.Hash)
			blockTxReceipt.ETHReceipt.ContractAddress = ethcom.HexToAddress(tx.Get("receipt_contract_address").String())
			blockTxReceipt.ETHReceipt.CumulativeGasUsed = tx.Get("receipt_cumulative_gas_used").Uint()
			blockTxReceipt.ETHReceipt.GasUsed = tx.Get("receipt_gas_used").Uint()
			//blockTxReceipt.ETHReceipt.Logs = blockTx.Status
			blockTxReceipt.ETHReceipt.Logs = make([]*types.Log, 0)
			logs := tx.Get("logs").Array()
			if logs != nil && len(logs) > 0 {
				for _, logData := range logs {
					txLog := &types.Log{}
					txLog.Address = ethcom.HexToAddress(logData.Get("address").String())
					txLog.Data = ethcom.FromHex(logData.Get("data").String())
					txLog.BlockNumber = block.BlockHeight
					txLog.TxHash = ethcom.HexToHash(logData.Get("transaction_hash").String())
					txLog.TxIndex = uint(logData.Get("transaction_index").Uint())
					txLog.BlockHash = ethcom.HexToHash(logData.Get("block_hash").String())
					txLog.Index = uint(logData.Get("log_index").Uint())

					//处理Topics
					topic0 := logData.Get("topic0").String()
					topic1 := logData.Get("topic1").String()
					topic2 := logData.Get("topic2").String()
					topic3 := logData.Get("topic3").String()
					topics := make([]ethcom.Hash, 0)
					if len(topic0) > 0 {
						topics = append(topics, ethcom.HexToHash(topic0))
					}
					if len(topic1) > 0 {
						topics = append(topics, ethcom.HexToHash(topic1))
					}
					if len(topic2) > 0 {
						topics = append(topics, ethcom.HexToHash(topic2))
					}
					if len(topic3) > 0 {
						topics = append(topics, ethcom.HexToHash(topic3))
					}
					txLog.Topics = topics

					blockTxReceipt.ETHReceipt.Logs = append(blockTxReceipt.ETHReceipt.Logs, txLog)
				}
			}

			//最后把ETHReceipt转为raw字符串
			raw, _ := json.Marshal(blockTxReceipt.ETHReceipt)
			blockTxReceipt.Raw = string(raw)
			blockTx.Receipt = blockTxReceipt

			blockTx.InternalTxs = make([]*BlockTransaction, 0)

			//把内部交易的原生币交易也加入到交易数组
			interTxs := tx.Get("internal_transactions").Array()
			if interTxs != nil && len(interTxs) > 0 {
				for _, interTx := range interTxs {
					value, _ := decimal.NewFromString(interTx.Get("value").String())
					txType := interTx.Get("type").String()
					if !value.IsZero() && txType != "DELEGATECALL" {

						blockInterTx := &BlockTransaction{}
						blockInterTx.Hash = blockTx.Hash
						blockInterTx.BlockNumber = blockTx.BlockNumber
						blockInterTx.BlockHash = blockTx.BlockHash
						blockInterTx.From = interTx.Get("from").String()
						blockInterTx.To = interTx.Get("to").String()
						blockInterTx.Value = hexutil.EncodeBig(common.StringNumToBigIntWithExp(interTx.Get("value").String(), 0))
						blockInterTx.BlockHeight = block.BlockHeight
						blockInterTx.Status = blockTx.Status
						blockInterTx.Receipt = nil
						blockInterTx.Decimal = decimals

						//blockInterTxReceipt := &TransactionReceipt{ETHReceipt: &types.Receipt{}}
						//blockInterTxReceipt.ETHReceipt.Status = blockTx.Status
						//blockInterTxReceipt.ETHReceipt.TxHash = ethcom.HexToHash(blockTx.Hash)
						//blockInterTxReceipt.ETHReceipt.Logs = make([]*types.Log, 0)
						////最后把ETHReceipt转为raw字符串
						//blockInterTxReceiptRaw, _ := json.Marshal(blockInterTxReceipt.ETHReceipt)
						//blockInterTxReceipt.Raw = string(blockInterTxReceiptRaw)
						//blockInterTx.Receipt = blockInterTxReceipt

						//添加内部交易
						//blockTxs = append(blockTxs, blockInterTx)
						blockTx.InternalTxs = append(blockTx.InternalTxs, blockInterTx)

					}

				}
			}

			//主交易加入到数组
			blockTxs = append(blockTxs, blockTx)
		}
	}
	block.Transactions = blockTxs
	return block
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
	TokenName       string
	TokenSymbol     string
	TokenDecimals   uint8
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

		if len(log.Topics) != 3 {
			continue
		}

		event, _ := ERC20_ABI.EventByID(log.Topics[0])
		if event == nil || event.Name != "Transfer" {
			continue
		}

		address := strings.ToLower(log.Address.String())

		var transfer TransferEvent
		err = bc.UnpackLog(&transfer, "Transfer", *log)
		if err != nil {
			continue
		}

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
	Status           uint64 `json:"-"`
	Receipt          *TransactionReceipt
	Decimal          int32
	InternalTxs      []*BlockTransaction `json:"-"` //内部交易数组
}

func (this *BlockTransaction) GetAmountEthString() string {
	amount, _ := hexutil.DecodeBig(this.Value)
	amountVal := common.BigIntToDecimals(amount, this.Decimal)
	return amountVal.String()
}

func (this *BlockTransaction) GetTxFeeEthString() string {
	return this.GetTxFeeValue().String()
}

func (this *BlockTransaction) GetTxFeeValue() decimal.Decimal {
	// gas无空值，手续费为0
	if len(this.GasPrice) == 0 {
		return decimal.Zero
	}
	gasPrice, _ := hexutil.DecodeBig(this.GasPrice)
	gas := common.StringNumToBigIntWithExp(this.Gas, 0)
	fee := big.NewInt(0)
	fee.Mul(gasPrice, gas)
	feeprice := common.BigIntToDecimals(fee, this.Decimal)
	return feeprice
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

//type CallMsg struct {
//	From     string `json:"from"`
//	To       string `json:"to"`
//	Data     string `json:"data"`
//	Value    string `json:"value"`
//	gas      string `json:"gas"`
//	gasPrice string `json:"gasPrice"`
//}

type CallMsg struct {
	To       ethcom.Address `json:"to"`
	From     ethcom.Address `json:"from"`
	Nonce    uint64         `json:"nonce"`
	Value    *big.Int       `json:"value"`
	GasLimit uint64         `json:"gasLimit"`
	Gas      uint64         `json:"gas"`
	GasPrice *big.Int       `json:"gasPrice"`
	Data     []byte         `json:"data"`
}

func (msg *CallMsg) UnmarshalJSON(data []byte) error {
	obj := gjson.ParseBytes(data)
	msg.From = ethcom.HexToAddress(obj.Get("from").String())
	msg.To = ethcom.HexToAddress(obj.Get("to").String())
	msg.Nonce, _ = hexutil.DecodeUint64(obj.Get("nonce").String())
	msg.Value, _ = hexutil.DecodeBig(obj.Get("value").String())
	msg.GasLimit, _ = hexutil.DecodeUint64(obj.Get("gasLimit").String())
	msg.Gas, _ = hexutil.DecodeUint64(obj.Get("gas").String())
	msg.GasPrice, _ = hexutil.DecodeBig(obj.Get("gasPrice").String())
	msg.Data, _ = hexutil.Decode(obj.Get("data").String())
	return nil
}

func (msg *CallMsg) MarshalJSON() ([]byte, error) {
	obj := map[string]interface{}{
		"from":     msg.From.String(),
		"to":       msg.To.String(),
		"nonce":    hexutil.EncodeUint64(msg.Nonce),
		"gasLimit": hexutil.EncodeUint64(msg.GasLimit),
		"gas":      hexutil.EncodeUint64(msg.Gas),
	}

	if msg.Value != nil {
		obj["value"] = hexutil.EncodeBig(msg.Value)
	}
	if msg.GasPrice != nil {
		obj["gasPrice"] = hexutil.EncodeBig(msg.GasPrice)
	}
	if msg.Data != nil {
		obj["data"] = hexutil.Encode(msg.Data)
	}
	return json.Marshal(obj)
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

type AddressSet struct {
	m map[string]bool
	sync.RWMutex
}

func NewAddressSet() *AddressSet {
	return &AddressSet{
		m: map[string]bool{},
	}
}
func (s *AddressSet) Add(item string) {
	s.Lock()
	defer s.Unlock()
	s.m[item] = true
}
func (s *AddressSet) Remove(item string) {
	s.Lock()
	s.Unlock()
	delete(s.m, item)
}
func (s *AddressSet) Has(item string) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.m[item]
	return ok
}
func (s *AddressSet) Len() int {
	return len(s.List())
}
func (s *AddressSet) Clear() {
	s.Lock()
	defer s.Unlock()
	s.m = map[string]bool{}
}
func (s *AddressSet) IsEmpty() bool {
	if s.Len() == 0 {
		return true
	}
	return false
}
func (s *AddressSet) List() []string {
	s.RLock()
	defer s.RUnlock()
	list := []string{}
	for item := range s.m {
		list = append(list, item)
	}
	return list
}
