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
	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/v2/common"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/blocktree/quorum-adapter/quorum_addrdec"
	"github.com/blocktree/quorum-adapter/quorum_rpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcom "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"reflect"

	//	"log"
	"math/big"
	"strings"
)

type WalletManager struct {
	openwallet.AssetsAdapterBase

	WalletClient            *quorum_rpc.Client              // 节点客户端
	Config                  *WalletConfig                   //钱包管理配置
	Blockscanner            openwallet.BlockScanner         //区块扫描器
	Decoder                 openwallet.AddressDecoderV2     //地址编码器
	TxDecoder               openwallet.TransactionDecoder   //交易单编码器
	ContractDecoder         openwallet.SmartContractDecoder //智能合约解释器
	Log                     *log.OWLogger                   //日志工具
	CustomAddressEncodeFunc func(address string) string     //自定义地址转换算法
	CustomAddressDecodeFunc func(address string) string     //自定义地址转换算法
}

func NewWalletManager() *WalletManager {
	wm := WalletManager{}
	wm.Config = NewConfig(Symbol)
	wm.Blockscanner = NewBlockScanner(&wm)
	wm.Decoder = &quorum_addrdec.Default
	wm.TxDecoder = NewTransactionDecoder(&wm)
	wm.ContractDecoder = &EthContractDecoder{wm: &wm}
	wm.Log = log.NewOWLogger(wm.Symbol())
	wm.CustomAddressEncodeFunc = CustomAddressEncode
	wm.CustomAddressDecodeFunc = CustomAddressDecode

	return &wm
}

func (wm *WalletManager) GetTransactionCount(addr string) (uint64, error) {
	addr = wm.CustomAddressDecodeFunc(addr)
	params := []interface{}{
		AppendOxToAddress(addr),
		"latest",
	}

	if wm.WalletClient == nil {
		return 0, fmt.Errorf("wallet client is not initialized")
	}

	result, err := wm.WalletClient.Call("eth_getTransactionCount", params)
	if err != nil {
		return 0, err
	}

	nonceStr := result.String()
	return hexutil.DecodeUint64(nonceStr)
}

func (wm *WalletManager) GetTransactionReceipt(transactionId string) (*TransactionReceipt, error) {
	params := []interface{}{
		transactionId,
	}

	var ethReceipt types.Receipt
	result, err := wm.WalletClient.Call("eth_getTransactionReceipt", params)
	if err != nil {
		return nil, err
	}

	err = ethReceipt.UnmarshalJSON([]byte(result.Raw))
	if err != nil {
		return nil, err
	}

	txReceipt := &TransactionReceipt{ETHReceipt: &ethReceipt, Raw: result.Raw}

	return txReceipt, nil

}

func (wm *WalletManager) GetTransactionByHash(txid string) (*BlockTransaction, error) {
	params := []interface{}{
		AppendOxToAddress(txid),
	}

	var tx BlockTransaction
	result, err := wm.WalletClient.Call("eth_getTransactionByHash", params)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(result.Raw), &tx)
	if err != nil {
		return nil, err
	}
	blockHeight, err := hexutil.DecodeUint64(tx.BlockNumber)
	if err != nil {
		blockHeight = 0
	}
	tx.BlockHeight = blockHeight
	tx.From = wm.CustomAddressEncodeFunc(tx.From)
	tx.To = wm.CustomAddressEncodeFunc(tx.To)
	return &tx, nil
}

func (wm *WalletManager) GetBlockByNum(blockNum uint64, showTransactionSpec bool) (*EthBlock, error) {
	params := []interface{}{
		hexutil.EncodeUint64(blockNum),
		showTransactionSpec,
	}
	var ethBlock EthBlock

	result, err := wm.WalletClient.Call("eth_getBlockByNumber", params)
	if err != nil {
		return nil, err
	}

	if showTransactionSpec {
		err = json.Unmarshal([]byte(result.Raw), &ethBlock)
	} else {
		err = json.Unmarshal([]byte(result.Raw), &ethBlock.BlockHeader)
	}
	if err != nil {
		return nil, err
	}
	ethBlock.BlockHeight, err = hexutil.DecodeUint64(ethBlock.BlockNumber)
	if err != nil {
		return nil, err
	}
	return &ethBlock, nil
}

func (wm *WalletManager) RecoverUnscannedTransactions(unscannedTxs []*openwallet.UnscanRecord) ([]*BlockTransaction, error) {
	allTxs := make([]*BlockTransaction, 0, len(unscannedTxs))
	for _, unscanned := range unscannedTxs {
		tx, err := wm.GetTransactionByHash(unscanned.TxID)
		if err != nil {
			return nil, err
		}
		allTxs = append(allTxs, tx)
	}
	return allTxs, nil
}

// ERC20GetAddressBalance
func (wm *WalletManager) ERC20GetAddressBalance(address string, contractAddr string) (*big.Int, error) {

	address = wm.CustomAddressDecodeFunc(address)
	contractAddr = wm.CustomAddressDecodeFunc(contractAddr)
	address = AppendOxToAddress(address)
	contractAddr = AppendOxToAddress(contractAddr)

	//abi编码
	data, err := wm.EncodeABIParam(ERC20_ABI, "balanceOf", address)
	if err != nil {
		return nil, err
	}

	//toAddr := ethcom.HexToAddress(contractAddr)
	callMsg := CallMsg{
		From: address,
		To:   contractAddr,
		Data: hexutil.Encode(data),
	}

	result, err := wm.EthCall(callMsg, "latest")
	if err != nil {
		return nil, err
	}

	rMap, _, err := wm.DecodeABIResult(ERC20_ABI, "balanceOf", result)
	if err != nil {
		return nil, err
	}
	balance, ok := rMap[""].(*big.Int)
	if !ok {
		return big.NewInt(0), fmt.Errorf("balance type is not big.Int ")
	}
	return balance, nil

}

// GetAddrBalance
func (wm *WalletManager) GetAddrBalance(address string, sign string) (*big.Int, error) {
	address = wm.CustomAddressDecodeFunc(address)
	params := []interface{}{
		AppendOxToAddress(address),
		sign,
	}
	result, err := wm.WalletClient.Call("eth_getBalance", params)
	if err != nil {
		return big.NewInt(0), err
	}

	balance, err := hexutil.DecodeBig(result.String())
	if err != nil {
		return big.NewInt(0), err
	}
	return balance, nil
}

// GetBlockNumber
func (wm *WalletManager) GetBlockNumber() (uint64, error) {
	param := make([]interface{}, 0)
	result, err := wm.WalletClient.Call("eth_blockNumber", param)
	if err != nil {
		return 0, err
	}
	return hexutil.DecodeUint64(result.String())
}

func (wm *WalletManager) GetTransactionFeeEstimated(from string, to string, value *big.Int, data []byte) (*txFeeInfo, error) {

	var (
		gasLimit *big.Int
		gasPrice *big.Int
		err      error
	)
	if wm.Config.FixGasLimit.Cmp(big.NewInt(0)) > 0 {
		//配置设置固定gasLimit
		gasLimit = wm.Config.FixGasLimit
	} else {
		//动态计算gas消耗

		gasLimit, err = wm.GetGasEstimated(from, to, value, data)
		if err != nil {
			return nil, err
		}
	}

	if wm.Config.FixGasPrice.Cmp(big.NewInt(0)) > 0 {
		//配置设置固定gasLimit
		gasPrice = wm.Config.FixGasPrice
	} else {
		//动态计算gasPrice

		gasPrice, err = wm.GetGasPrice()
		if err != nil {
			return nil, err
		}
	}

	//	fee := new(big.Int)
	//	fee.Mul(gasLimit, gasPrice)

	feeInfo := &txFeeInfo{
		GasLimit: gasLimit,
		GasPrice: gasPrice,
		//		Fee:      fee,
	}

	feeInfo.CalcFee()
	return feeInfo, nil
}

// GetGasEstimated
func (wm *WalletManager) GetGasEstimated(from string, to string, value *big.Int, data []byte) (*big.Int, error) {
	//toAddr := ethcom.HexToAddress(to)
	callMsg := map[string]interface{}{
		"from": wm.CustomAddressDecodeFunc(from),
		"to":   wm.CustomAddressDecodeFunc(to),
		"data": hexutil.Encode(data),
	}

	result, err := wm.WalletClient.Call("eth_estimateGas", []interface{}{callMsg})
	if err != nil {
		return big.NewInt(0), err
	}
	gasLimit, err := common.StringValueToBigInt(result.String(), 16)
	if err != nil {
		return big.NewInt(0), fmt.Errorf("convert estimated gas[%v] format to bigint failed, err = %v\n", result.String(), err)
	}
	return gasLimit, nil
}

func (wm *WalletManager) GetGasPrice() (*big.Int, error) {

	result, err := wm.WalletClient.Call("eth_gasPrice", []interface{}{})
	if err != nil {
		return big.NewInt(0), err
	}

	gasLimit, err := common.StringValueToBigInt(result.String(), 16)
	if err != nil {
		return big.NewInt(0), fmt.Errorf("convert estimated gas[%v] format to bigint failed, err = %v\n", result.String(), err)
	}
	return gasLimit, nil
}

func (wm *WalletManager) SetNetworkChainID() (uint64, error) {

	result, err := wm.WalletClient.Call("eth_chainId", nil)
	if err != nil {
		return 0, err
	}
	id, err := hexutil.DecodeUint64(result.String())
	if err != nil {
		return 0, err
	}
	wm.Config.ChainID = id
	//wm.Log.Debugf("Network chainID: %d", wm.Config.ChainID)
	return id, nil
}

// EncodeABIParam 编码API调用参数
func (wm *WalletManager) EncodeABIParam(abiInstance abi.ABI, abiParam ...string) ([]byte, error) {

	var (
		err  error
		args = make([]interface{}, 0)
	)

	if len(abiParam) == 0 {
		return nil, fmt.Errorf("abi param length is empty")
	}
	method := abiParam[0]
	//转化string参数为abi调用参数
	abiMethod, ok := abiInstance.Methods[method]
	if !ok {
		return nil, fmt.Errorf("abi method can not found")
	}
	abiArgs := abiParam[1:]
	if len(abiMethod.Inputs) != len(abiArgs) {
		return nil, fmt.Errorf("abi input arguments is: %d, except is : %d", len(abiArgs), len(abiMethod.Inputs))
	}
	for i, input := range abiMethod.Inputs {
		var a interface{}
		switch input.Type.T {
		case abi.BoolTy:
			a = common.NewString(abiArgs[i]).Bool()
		case abi.UintTy, abi.IntTy:
			a, err = convertParamToNum(abiArgs[i], input.Type.Kind)
		case abi.AddressTy:
			a = ethcom.HexToAddress(AppendOxToAddress(abiArgs[i]))
		case abi.FixedBytesTy, abi.BytesTy, abi.HashTy:
			slice, decodeErr := hexutil.Decode(AppendOxToAddress(abiArgs[i]))
			if decodeErr != nil {
				slice = owcrypt.Hash([]byte(abiArgs[i]), 0, owcrypt.HASH_ALG_KECCAK256)
				//return nil, fmt.Errorf("abi input hex string can not convert byte, err: %v", decodeErr)
			}
			var fixBytes [32]byte
			copy(fixBytes[:], slice)
			a = fixBytes
		case abi.StringTy:
			a = abiArgs[i]
		}
		if err != nil {
			return nil, err
		}
		args = append(args, a)
	}

	return abiInstance.Pack(method, args...)
}

// DecodeABIResult 解码ABI结果
func (wm *WalletManager) DecodeABIResult(abiInstance abi.ABI, method string, dataHex string) (map[string]interface{}, string, error) {

	var (
		err        error
		resultJSON []byte
		result     = make(CallResult)
	)
	data, _ := hexutil.Decode(dataHex)
	if len(data) == 0 {
		return result, "", nil
	}

	err = abiInstance.UnpackIntoMap(result, method, data)
	if err != nil {
		return result, string(resultJSON), err
	}
	resultJSON, err = result.MarshalJSON()
	return result, string(resultJSON), err
}

// DecodeReceiptLogResult 解码回执日志结果
func (wm *WalletManager) DecodeReceiptLogResult(abiInstance abi.ABI, log types.Log) (map[string]interface{}, string, string, error) {

	var (
		err        error
		resultJSON []byte
		result     = make(CallResult)
		event      *abi.Event
	)

	bc := bind.NewBoundContract(ethcom.HexToAddress("0x0"), abiInstance, nil, nil, nil)
	//wm.Log.Debugf("log.txid: %s", log.TxHash.String())
	//wm.Log.Debugf("log.Topics[0]: %s", log.Topics[0].Hex())
	//for _, e := range abiInstance.Events {
	//	wm.Log.Debugf("event: %s, ID: %s", e.Name, e.ID().Hex())
	//}
	event, err = abiInstance.EventByID(log.Topics[0])
	if err != nil {
		return result, "", "", err
	}
	err = bc.UnpackLogIntoMap(result, event.Name, log)
	if err != nil {
		return result, "", "", err
	}
	resultJSON, err = result.MarshalJSON()
	return result, event.Name, string(resultJSON), err
}

func (wm *WalletManager) EthCall(callMsg CallMsg, sign string) (string, error) {
	param := map[string]interface{}{
		"from":  wm.CustomAddressDecodeFunc(callMsg.From),
		"to":    wm.CustomAddressDecodeFunc(callMsg.To),
		"value": callMsg.Value,
		"data":  callMsg.Data,
	}
	result, err := wm.WalletClient.Call("eth_call", []interface{}{param, sign})
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

// SendRawTransaction
func (wm *WalletManager) SendRawTransaction(signedTx string) (string, error) {
	params := []interface{}{
		signedTx,
	}

	result, err := wm.WalletClient.Call("eth_sendRawTransaction", params)
	if err != nil {
		return "", err
	}

	return result.String(), nil
}

// IsContract 是否合约
func (wm *WalletManager) IsContract(address string) (bool, error) {
	params := []interface{}{
		wm.CustomAddressDecodeFunc(address),
		"latest",
	}

	result, err := wm.WalletClient.Call("eth_getCode", params)
	if err != nil {
		return false, err
	}

	if result.String() == "0x" {
		return false, nil
	} else {
		return true, nil
	}

}

// GetAddressNonce
func (wm *WalletManager) GetAddressNonce(wrapper openwallet.WalletDAI, address string) uint64 {
	var (
		key           = wm.Symbol() + "-nonce"
		nonce         uint64
		nonce_db      interface{}
		nonce_onchain uint64
		err           error
	)

	//获取db记录的nonce并确认nonce值
	nonce_db, _ = wrapper.GetAddressExtParam(address, key)

	//判断nonce_db是否为空,为空则说明当前nonce是0
	if nonce_db == nil {
		nonce = 0
	} else {
		nonce = common.NewString(nonce_db).UInt64()
	}

	nonce_onchain, err = wm.GetTransactionCount(address)
	if err != nil {
		return nonce
	}

	//如果本地nonce_db > 链上nonce,采用本地nonce,否则采用链上nonce
	if nonce > nonce_onchain {
		//wm.Log.Debugf("%s nonce_db=%v > nonce_chain=%v,Use nonce_db...", address, nonce_db, nonce_onchain)
	} else {
		nonce = nonce_onchain
		//wm.Log.Debugf("%s nonce_db=%v <= nonce_chain=%v,Use nonce_chain...", address, nonce_db, nonce_onchain)
	}

	return nonce
}

// UpdateAddressNonce
func (wm *WalletManager) UpdateAddressNonce(wrapper openwallet.WalletDAI, address string, nonce uint64) {
	key := wm.Symbol() + "-nonce"
	err := wrapper.SetAddressExtParam(address, key, nonce)
	if err != nil {
		wm.Log.Errorf("WalletDAI SetAddressExtParam failed, err: %v", err)
	}
}

func AppendOxToAddress(addr string) string {
	if strings.Index(addr, "0x") == -1 {
		return "0x" + addr
	}
	return addr
}

func removeOxFromHex(value string) string {
	result := value
	if strings.Index(value, "0x") != -1 {
		result = common.Substr(value, 2, len(value))
	}
	return result
}

func convertParamToNum(param string, kind reflect.Kind) (interface{}, error) {
	var (
		base int
		bInt *big.Int
		err  error
	)
	if strings.HasPrefix(param, "0x") {
		base = 16
	} else {
		base = 10
	}
	bInt, err = common.StringValueToBigInt(param, base)
	if err != nil {
		return nil, err
	}

	switch kind {
	case reflect.Uint:
		return uint(bInt.Uint64()), nil
	case reflect.Uint8:
		return uint8(bInt.Uint64()), nil
	case reflect.Uint16:
		return uint16(bInt.Uint64()), nil
	case reflect.Uint32:
		return uint32(bInt.Uint64()), nil
	case reflect.Uint64:
		return uint64(bInt.Uint64()), nil
	case reflect.Int:
		return int(bInt.Int64()), nil
	case reflect.Int8:
		return int8(bInt.Int64()), nil
	case reflect.Int16:
		return int16(bInt.Int64()), nil
	case reflect.Int32:
		return int32(bInt.Int64()), nil
	case reflect.Int64:
		return int64(bInt.Int64()), nil
	case reflect.Ptr:
		return bInt, nil
	default:
		return nil, fmt.Errorf("abi input arguments: %v is invaild integer type", param)
	}
}

func CustomAddressEncode(address string) string {
	return address
}
func CustomAddressDecode(address string) string {
	return address
}
