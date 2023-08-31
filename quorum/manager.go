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
	"github.com/tidwall/gjson"

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
	"github.com/ethereum/go-ethereum/ethclient"

	//	"log"
	"math/big"
	"strings"
)

type WalletManager struct {
	openwallet.AssetsAdapterBase

	RawClient               *ethclient.Client               //原生ETH客户端
	WalletClient            *quorum_rpc.Client              // 节点客户端
	BroadcastClient         *quorum_rpc.Client              // 节点客户端
	Config                  *WalletConfig                   //钱包管理配置
	Blockscanner            openwallet.BlockScanner         //区块扫描器
	Decoder                 openwallet.AddressDecoderV2     //地址编码器
	TxDecoder               openwallet.TransactionDecoder   //交易单编码器
	ContractDecoder         openwallet.SmartContractDecoder //智能合约解释器
	NFTContractDecoder      openwallet.NFTContractDecoder   //NFT智能合约解释器
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
	wm.NFTContractDecoder = &NFTContractDecoder{wm: &wm}
	wm.Log = log.NewOWLogger(wm.Symbol())
	wm.CustomAddressEncodeFunc = CustomAddressEncode
	wm.CustomAddressDecodeFunc = CustomAddressDecode

	return &wm
}

func NewWalletManagerWithSymbol(symbol string) *WalletManager {
	wm := WalletManager{}
	wm.Config = NewConfig(symbol)
	wm.Blockscanner = NewBlockScanner(&wm)
	wm.Decoder = &quorum_addrdec.Default
	wm.TxDecoder = NewTransactionDecoder(&wm)
	wm.ContractDecoder = &EthContractDecoder{wm: &wm}
	wm.NFTContractDecoder = &NFTContractDecoder{wm: &wm}
	wm.Log = log.NewOWLogger(symbol)
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

func (wm *WalletManager) GetETHBlockByNum(blockNum uint64, showTransactionSpec bool) (*EthBlock, error) {
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

func (wm *WalletManager) GetBlockByNum(blockNum uint64, showTransactionSpec bool) (*EthBlock, error) {
	if wm.Config.UseQNSingleFlightRPC == 1 && showTransactionSpec {
		return wm.GetQNBlockWithReceipts(blockNum)
	}

	return wm.GetETHBlockByNum(blockNum, showTransactionSpec)
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
		From:  ethcom.HexToAddress(address),
		To:    ethcom.HexToAddress(contractAddr),
		Data:  data,
		Value: big.NewInt(0),
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
		gasPrice.Add(gasPrice, wm.Config.OffsetsGasPrice)
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

	if value != nil {
		callMsg["value"] = hexutil.EncodeBig(value)
	}

	result, err := wm.WalletClient.Call("eth_estimateGas", []interface{}{callMsg})
	if err != nil {
		return big.NewInt(0), err
	}
	gasLimit, err := common.StringValueToBigInt(result.String(), 16)
	if err != nil {
		return big.NewInt(0), fmt.Errorf("convert estimated gas[%v] format to bigint failed, err = %v\n", result.String(), err)
	}
	if data != nil {
		//当data有值时，代表交易是调用合约， gasLimit = gasLimit * 1.1，确保gas足够
		gasLimit = gasLimit.Mul(gasLimit, big.NewInt(110))
		gasLimit = gasLimit.Div(gasLimit, big.NewInt(100))
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
		//var a interface{}
		a, err := convertStringParamToABIParam(input.Type, abiArgs[i])
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
	if callMsg.Value == nil {
		callMsg.Value = big.NewInt(0)
	}
	param := map[string]interface{}{
		"from":  callMsg.From.String(),
		"to":    callMsg.To.String(),
		"value": hexutil.EncodeBig(callMsg.Value),
		"data":  hexutil.Encode(callMsg.Data),
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

	//NonceComputeMode = 0时，使用外部系统的自增值
	if wm.Config.NonceComputeMode == 0 {
		//获取db记录的nonce并确认nonce值
		nonce_db, _ = wrapper.GetAddressExtParam(address, key)

		//判断nonce_db是否为空,为空则说明当前nonce是0
		if nonce_db == nil {
			nonce = 0
		} else {
			nonce = common.NewString(nonce_db).UInt64()
		}
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

	//wm.Log.Debugf("nonce: %v", nonce)

	return nonce
}

func (wm *WalletManager) CallABI(contractAddr string, abiInstance abi.ABI, abiParam ...string) (map[string]interface{}, *openwallet.Error) {

	methodName := ""
	if len(abiParam) > 0 {
		methodName = abiParam[0]
	}

	//abi编码
	data, err := wm.EncodeABIParam(abiInstance, abiParam...)
	if err != nil {
		return nil, openwallet.ConvertError(err)
	}

	callMsg := CallMsg{
		From:  ethcom.HexToAddress("0x00"),
		To:    ethcom.HexToAddress(contractAddr),
		Data:  data,
		Value: big.NewInt(0),
	}

	result, err := wm.EthCall(callMsg, "latest")
	if err != nil {
		return nil, openwallet.ConvertError(err)
	}

	rMap, _, err := wm.DecodeABIResult(abiInstance, methodName, result)
	if err != nil {
		return nil, openwallet.ConvertError(err)
	}

	return rMap, nil
}

func (wm *WalletManager) SupportsInterface(contractAddr string) string {
	//is support erc721
	result721, _ := wm.CallABI(contractAddr, ERC721_ABI, "supportsInterface", "0x80ac58cd")
	//if err != nil {
	//	log.Errorf("SupportsInterface: %+v", err)
	//}
	//is support erc1155
	result1155, _ := wm.CallABI(contractAddr, ERC721_ABI, "supportsInterface", "0xd9b67a26")
	//if err != nil {
	//	log.Errorf("SupportsInterface: %+v", err)
	//}

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

// UpdateAddressNonce
func (wm *WalletManager) UpdateAddressNonce(wrapper openwallet.WalletDAI, address string, nonce uint64) {
	key := wm.Symbol() + "-nonce"
	err := wrapper.SetAddressExtParam(address, key, nonce)
	if err != nil {
		wm.Log.Errorf("WalletDAI SetAddressExtParam failed, err: %v", err)
	}
}

// LoadContractInfo 通过地址加载合约信息
func (wm *WalletManager) LoadContractInfo(addr string) *openwallet.SmartContract {
	var (
		contract *openwallet.SmartContract
		abiInst  abi.ABI
		abiJSON  = ""
		token    = ""
		name     = ""
	)
	inferfaceType := wm.SupportsInterface(addr)
	switch inferfaceType {
	case openwallet.InterfaceTypeERC721:
		abiJSON = ERC721_ABI_JSON
		abiInst = ERC721_ABI

	case openwallet.InterfaceTypeERC1155:
		abiJSON = ERC1155_ABI_JSON
		abiInst = ERC1155_ABI
	default:
		return nil
	}

	result, err := wm.CallABI(addr, abiInst, "symbol")
	if err == nil {
		v, ok := result[""].(string)
		if ok {
			token = v
		}
	}
	result, err = wm.CallABI(addr, abiInst, "name")
	if err == nil {
		v, ok := result[""].(string)
		if ok {
			name = v
		}
	}

	contractId := openwallet.GenContractID(wm.Symbol(), addr)
	contract = &openwallet.SmartContract{
		ContractID: contractId,
		Symbol:     wm.Symbol(),
		Address:    addr,
		Decimals:   0,
		Token:      token,
		Name:       name,
		Protocol:   inferfaceType,
	}
	contract.SetABI(abiJSON)
	return contract
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

// convertStringParamToABIParam string参数转为ABI参数
func convertStringParamToABIParam(inputType abi.Type, abiArg string) (interface{}, error) {
	var (
		err error
		a   interface{}
	)

	switch inputType.T {
	case abi.BoolTy:
		a = common.NewString(abiArg).Bool()
	case abi.UintTy, abi.IntTy:
		a, err = convertParamToNum(abiArg, inputType)
	case abi.AddressTy:
		a = ethcom.HexToAddress(AppendOxToAddress(abiArg))
	case abi.FixedBytesTy, abi.BytesTy, abi.HashTy:
		slice, decodeErr := hexutil.Decode(AppendOxToAddress(abiArg))
		if decodeErr != nil {
			slice = owcrypt.Hash([]byte(abiArg), 0, owcrypt.HASH_ALG_KECCAK256)
			//return nil, fmt.Errorf("abi input hex string can not convert byte, err: %v", decodeErr)
		}
		// var fixBytes [32]byte
		// copy(fixBytes[:], slice)
		// a = fixBytes
		a = packFixArray(slice, a)
	case abi.StringTy:
		a = abiArg
	case abi.ArrayTy, abi.SliceTy:
		subArgs := strings.Split(abiArg, ",")
		a, err = convertArrayParamToABIParam(*inputType.Elem, subArgs)
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// convertArrayParamToABIParam 数组参数转化
func convertArrayParamToABIParam(inputType abi.Type, subArgs []string) (interface{}, error) {
	var (
		err error
		a   interface{}
	)

	switch inputType.T {
	case abi.BoolTy:
		arr := make([]bool, 0)
		for _, subArg := range subArgs {
			elem, subErr := convertStringParamToABIParam(inputType, subArg)
			if subErr != nil {
				err = subErr
				break
			}
			arr = append(arr, elem.(bool))
		}
		a = arr
	case abi.UintTy:
		arr := make([]uint, 0)
		for _, subArg := range subArgs {
			elem, subErr := convertStringParamToABIParam(inputType, subArg)
			if subErr != nil {
				err = subErr
				break
			}
			arr = append(arr, elem.(uint))
		}
		a = arr
	case abi.IntTy:
		arr := make([]int, 0)
		for _, subArg := range subArgs {
			elem, subErr := convertStringParamToABIParam(inputType, subArg)
			if subErr != nil {
				err = subErr
				break
			}
			arr = append(arr, elem.(int))
		}
		a = arr
	case abi.AddressTy:
		arr := make([]ethcom.Address, 0)
		for _, subArg := range subArgs {
			elem, subErr := convertStringParamToABIParam(inputType, subArg)
			if subErr != nil {
				err = subErr
				break
			}
			arr = append(arr, elem.(ethcom.Address))
		}
		a = arr
	case abi.FixedBytesTy, abi.BytesTy, abi.HashTy:
		arr := make([][32]byte, 0)
		for _, subArg := range subArgs {
			elem, subErr := convertStringParamToABIParam(inputType, subArg)
			if subErr != nil {
				err = subErr
				break
			}
			arr = append(arr, elem.([32]byte))
		}
		a = arr
	case abi.StringTy:
		arr := make([]string, 0)
		for _, subArg := range subArgs {
			elem, subErr := convertStringParamToABIParam(inputType, subArg)
			if subErr != nil {
				err = subErr
				break
			}
			arr = append(arr, elem.(string))
		}
		a = arr
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func convertParamToNum(param string, abiType abi.Type) (interface{}, error) {
	var (
		base int
		//bInt *big.Int
		//err  error
	)
	if strings.HasPrefix(param, "0x") {
		base = 16
	} else {
		base = 10
	}
	return common.StringValueToBigInt(param, base)
	//bInt, err = common.StringValueToBigInt(param, base)
	//if err != nil {
	//	return nil, err
	//}
	//
	//switch abiType.Kind {
	//case reflect.Uint:
	//	return uint(bInt.Uint64()), nil
	//case reflect.Uint8:
	//	return uint8(bInt.Uint64()), nil
	//case reflect.Uint16:
	//	return uint16(bInt.Uint64()), nil
	//case reflect.Uint32:
	//	return uint32(bInt.Uint64()), nil
	//case reflect.Uint64:
	//	return uint64(bInt.Uint64()), nil
	//case reflect.Int:
	//	return int(bInt.Int64()), nil
	//case reflect.Int8:
	//	return int8(bInt.Int64()), nil
	//case reflect.Int16:
	//	return int16(bInt.Int64()), nil
	//case reflect.Int32:
	//	return int32(bInt.Int64()), nil
	//case reflect.Int64:
	//	return int64(bInt.Int64()), nil
	//case reflect.Ptr:
	//	return bInt, nil
	//default:
	//	return nil, fmt.Errorf("abi input arguments: %v is invaild integer type", param)
	//}
}

func CustomAddressEncode(address string) string {
	return address
}
func CustomAddressDecode(address string) string {
	return address
}

func packFixArray(slice []byte, a interface{}) interface{} {
	switch len(slice) {
	case 2:
		var fixBytes [2]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	case 4:
		var fixBytes [4]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	case 6:
		var fixBytes [6]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	case 8:
		var fixBytes [8]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	case 10:
		var fixBytes [10]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	case 12:
		var fixBytes [12]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	case 14:
		var fixBytes [14]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	case 16:
		var fixBytes [16]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	case 18:
		var fixBytes [18]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	case 20:
		var fixBytes [20]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	case 22:
		var fixBytes [22]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	case 24:
		var fixBytes [24]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	case 26:
		var fixBytes [26]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	case 28:
		var fixBytes [28]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	case 30:
		var fixBytes [30]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	case 32:
		var fixBytes [32]byte
		copy(fixBytes[:], slice)
		a = fixBytes
	}
	return a
}

func (wm *WalletManager) GetBlockchainSyncStatus() (*openwallet.BlockchainSyncStatus, error) {

	result, err := wm.WalletClient.Call("eth_syncing", nil)
	if err != nil {
		return nil, err
	}

	status := &openwallet.BlockchainSyncStatus{}

	if result.IsObject() {
		obj := gjson.ParseBytes([]byte(result.Raw))
		status.NetworkBlockHeight, _ = hexutil.DecodeUint64(obj.Get("highestBlock").String())
		status.CurrentBlockHeight, _ = hexutil.DecodeUint64(obj.Get("currentBlock").String())
		status.Syncing = true
	} else {
		status.Syncing = false
	}

	return status, nil
}

// GetQNBlockWithReceipts
func (wm *WalletManager) GetQNBlockWithReceipts(blockNum uint64) (*EthBlock, error) {
	params := []interface{}{
		hexutil.EncodeUint64(blockNum),
	}
	result, err := wm.WalletClient.Call("qn_getBlockWithReceipts", params)
	if err != nil {
		return nil, err
	}
	var ethBlock EthBlock
	err = json.Unmarshal([]byte(result.Get("block").Raw), &ethBlock)
	if err != nil {
		return nil, err
	}
	ethBlock.BlockHeight, err = hexutil.DecodeUint64(ethBlock.BlockNumber)
	if err != nil {
		return nil, err
	}

	// parsing receipt
	receiptsMap := make(map[string]*TransactionReceipt, 0)
	receipts := result.Get("receipts").Array()
	if receipts != nil && len(receipts) > 0 {
		for _, receipt := range receipts {
			var ethReceipt types.Receipt
			err = ethReceipt.UnmarshalJSON([]byte(receipt.Raw))
			if err != nil {
				return nil, err
			}
			receiptsMap[ethReceipt.TxHash.String()] = &TransactionReceipt{ETHReceipt: &ethReceipt, Raw: receipt.Raw}
		}

		for _, tx := range ethBlock.Transactions {
			txReceipt := receiptsMap[tx.Hash]
			tx.Receipt = txReceipt
			tx.Gas = common.NewString(txReceipt.ETHReceipt.GasUsed).String()
			tx.Status = txReceipt.ETHReceipt.Status
			tx.Decimal = wm.Decimal()
		}
	}

	return &ethBlock, nil
}
