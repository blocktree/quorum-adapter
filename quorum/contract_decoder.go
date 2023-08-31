package quorum

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/v2/common"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcom "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"strconv"
	"strings"
	"time"
)

type EthContractDecoder struct {
	*openwallet.SmartContractDecoderBase
	wm *WalletManager
}

type AddrBalance struct {
	Address      string
	Balance      *big.Int
	TokenBalance *big.Int
	Index        int
}

func (this *AddrBalance) SetTokenBalance(b *big.Int) {
	this.TokenBalance = b
}

func (this *AddrBalance) GetAddress() string {
	return this.Address
}

func (this *AddrBalance) ValidTokenBalance() bool {
	if this.Balance == nil {
		return false
	}
	return true
}

type AddrBalanceInf interface {
	SetTokenBalance(b *big.Int)
	GetAddress() string
	ValidTokenBalance() bool
}

func (decoder *EthContractDecoder) GetTokenBalanceByAddress(contract openwallet.SmartContract, address ...string) ([]*openwallet.TokenBalance, error) {
	threadControl := make(chan int, 20)
	defer close(threadControl)
	resultChan := make(chan *openwallet.TokenBalance, 1024)
	defer close(resultChan)
	done := make(chan int, 1)
	var tokenBalanceList []*openwallet.TokenBalance
	count := len(address)

	go func() {
		//		log.Debugf("in save thread.")
		for i := 0; i < count; i++ {
			balance := <-resultChan
			if balance != nil {
				tokenBalanceList = append(tokenBalanceList, balance)
			}
			//log.Debugf("got one balance.")
		}
		done <- 1
	}()

	queryBalance := func(address string) {
		threadControl <- 1
		var balance *openwallet.TokenBalance
		defer func() {
			resultChan <- balance
			<-threadControl
		}()

		//		log.Debugf("in query thread.")
		balanceConfirmed, err := decoder.wm.ERC20GetAddressBalance(address, contract.Address)
		if err != nil {
			return
		}
		balanceUnconfirmed := big.NewInt(0)
		balanceAll := balanceConfirmed
		bstr := common.BigIntToDecimals(balanceAll, int32(contract.Decimals))
		if err != nil {
			return
		}

		cbstr := common.BigIntToDecimals(balanceConfirmed, int32(contract.Decimals))
		if err != nil {
			return
		}

		ucbstr := common.BigIntToDecimals(balanceUnconfirmed, int32(contract.Decimals))
		if err != nil {
			return
		}

		balance = &openwallet.TokenBalance{
			Contract: &contract,
			Balance: &openwallet.Balance{
				Address:          address,
				Symbol:           contract.Symbol,
				Balance:          bstr.String(),
				ConfirmBalance:   cbstr.String(),
				UnconfirmBalance: ucbstr.String(),
			},
		}
	}

	for i := range address {
		go queryBalance(address[i])
	}

	<-done

	if len(tokenBalanceList) != count {
		log.Error("unknown errors occurred .")
		return nil, errors.New("unknown errors occurred ")
	}
	return tokenBalanceList, nil
}

func (decoder *EthContractDecoder) EncodeRawTransactionCallMsg(wrapper openwallet.WalletDAI, rawTx *openwallet.SmartContractRawTransaction) (*CallMsg, *abi.ABI, *openwallet.Error) {
	var (
		callMsg CallMsg
	)

	if !rawTx.Coin.IsContract {
		return nil, nil, openwallet.Errorf(openwallet.ErrContractCallMsgInvalid, "contract call msg invalid ")
	}

	value := common.StringNumToBigIntWithExp(rawTx.Value, decoder.wm.Decimal())

	if len(rawTx.Raw) > 0 {
		var decErr error
		//直接的数据请求
		switch rawTx.RawType {
		case openwallet.TxRawTypeHex:
			rawBytes := hexutil.MustDecode(AppendOxToAddress(rawTx.Raw))
			decErr = rlp.DecodeBytes(rawBytes, &callMsg)
		case openwallet.TxRawTypeJSON:
			decErr = json.Unmarshal([]byte(rawTx.Raw), &callMsg)
		case openwallet.TxRawTypeBase64:
			rawBytes, _ := base64.StdEncoding.DecodeString(rawTx.Raw)
			decErr = rlp.DecodeBytes(rawBytes, &callMsg)
		}
		if decErr != nil {
			return nil, nil, openwallet.Errorf(openwallet.ErrContractCallMsgInvalid, decErr.Error())
		}

		return &callMsg, nil, nil
	} else {
		abiJSON := rawTx.Coin.Contract.GetABI()
		if len(abiJSON) == 0 {
			return nil, nil, openwallet.Errorf(openwallet.ErrContractCallMsgInvalid, "abi json is empty")
		}
		abiInstance, err := abi.JSON(strings.NewReader(abiJSON))
		if err != nil {
			return nil, nil, openwallet.Errorf(openwallet.ErrContractCallMsgInvalid, err.Error())
		}
		data, encErr := decoder.wm.EncodeABIParam(abiInstance, rawTx.ABIParam...)
		if encErr != nil {
			return nil, nil, openwallet.Errorf(openwallet.ErrContractCallMsgInvalid, encErr.Error())
		}
		defAddress, getErr := decoder.GetAssetsAccountDefAddress(wrapper, rawTx.Account.AccountID)
		if getErr != nil {
			return nil, nil, getErr
		}
		//toAddr := ethcom.HexToAddress(rawTx.Coin.Contract.Address)
		callMsg = CallMsg{
			From:  ethcom.HexToAddress(decoder.wm.CustomAddressDecodeFunc(defAddress.Address)),
			To:    ethcom.HexToAddress(decoder.wm.CustomAddressDecodeFunc(rawTx.Coin.Contract.Address)),
			Data:  data,
			Value: value,
		}
		log.Infof("call data: %s", hex.EncodeToString(data))
		return &callMsg, &abiInstance, nil
	}
}

func (decoder *EthContractDecoder) GetAssetsAccountDefAddress(wrapper openwallet.WalletDAI, accountID string) (*openwallet.Address, *openwallet.Error) {
	//获取wallet
	addresses, err := wrapper.GetAddressList(0, 1,
		"AccountID", accountID)
	if err != nil {
		return nil, openwallet.NewError(openwallet.ErrAddressNotFound, err.Error())
	}

	if len(addresses) == 0 {
		return nil, openwallet.Errorf(openwallet.ErrAccountNotAddress, "[%s] have not addresses", accountID)
	}
	return addresses[0], nil
}

// 调用合约ABI方法
func (decoder *EthContractDecoder) CallSmartContractABI(wrapper openwallet.WalletDAI, rawTx *openwallet.SmartContractRawTransaction) (*openwallet.SmartContractCallResult, *openwallet.Error) {

	callMsg, abiInstance, encErr := decoder.EncodeRawTransactionCallMsg(wrapper, rawTx)
	if encErr != nil {
		return nil, encErr
	}

	methodName := ""
	if len(rawTx.ABIParam) > 0 {
		methodName = rawTx.ABIParam[0]
	}

	callResult := &openwallet.SmartContractCallResult{
		Method: methodName,
	}

	result, err := decoder.wm.EthCall(*callMsg, "latest")
	if err != nil {
		callResult.Status = openwallet.SmartContractCallResultStatusFail
		callResult.Exception = err.Error()
		return callResult, openwallet.ConvertError(err)
	}

	if abiInstance != nil {
		_, rJSON, err := decoder.wm.DecodeABIResult(*abiInstance, callResult.Method, result)
		if err != nil {
			return nil, openwallet.ConvertError(err)
		}
		callResult.Value = rJSON
	}

	callResult.RawHex = result
	callResult.Status = openwallet.SmartContractCallResultStatusSuccess

	return callResult, nil
}

// 创建原始交易单
func (decoder *EthContractDecoder) CreateSmartContractRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.SmartContractRawTransaction) *openwallet.Error {

	var (
		keySignList = make([]*openwallet.KeySignature, 0)
		fee         *txFeeInfo
		feeErr      error
	)

	callMsg, _, encErr := decoder.EncodeRawTransactionCallMsg(wrapper, rawTx)
	if encErr != nil {
		return encErr
	}

	data := callMsg.Data

	//amount := common.StringNumToBigIntWithExp(rawTx.Value, decoder.wm.Decimal())
	amount := callMsg.Value

	if callMsg.GasPrice != nil && callMsg.GasPrice.Cmp(big.NewInt(0)) > 0 && callMsg.Gas > 0 {
		bigGas := new(big.Int)
		bigGas.SetUint64(callMsg.Gas)
		fee = &txFeeInfo{
			GasLimit: bigGas,
			GasPrice: callMsg.GasPrice,
		}
		fee.CalcFee()
	} else {
		//计算手续费
		fee, feeErr = decoder.wm.GetTransactionFeeEstimated(
			strings.ToLower(callMsg.From.String()),
			strings.ToLower(callMsg.To.String()),
			amount, data)
		if feeErr != nil {
			//decoder.wm.Log.Std.Error("GetTransactionFeeEstimated from[%v] -> to[%v] failed, err=%v", callMsg.From, callMsg.To, createErr)
			return openwallet.Errorf(openwallet.ErrCreateRawSmartContractTransactionFailed, feeErr.Error())
		}

		if rawTx.FeeRate != "" {
			fee.GasPrice = common.StringNumToBigIntWithExp(rawTx.FeeRate, decoder.wm.Decimal()) //ConvertToBigInt(rawTx.FeeRate, 16)
			fee.CalcFee()
		}
	}

	//检查调用地址是否有足够手续费
	coinBalance, err := decoder.wm.GetAddrBalance(strings.ToLower(callMsg.From.String()), "pending")
	if err != nil {
		return openwallet.Errorf(openwallet.ErrCreateRawSmartContractTransactionFailed, err.Error())
	}

	if coinBalance.Cmp(fee.Fee) < 0 {
		coinBalance := common.BigIntToDecimals(coinBalance, decoder.wm.Decimal())
		return openwallet.Errorf(openwallet.ErrInsufficientFees, "the [%s] balance: %s is not enough to call smart contract", rawTx.Coin.Symbol, coinBalance.String())
	}

	gasprice := common.BigIntToDecimals(fee.GasPrice, decoder.wm.Decimal())
	totalFeeDecimal := common.BigIntToDecimals(fee.Fee, decoder.wm.Decimal())

	addr, err := wrapper.GetAddress(strings.ToLower(callMsg.From.String()))
	if err != nil {
		return openwallet.NewError(openwallet.ErrAccountNotAddress, err.Error())
	}

	nonce := decoder.wm.GetAddressNonce(wrapper, strings.ToLower(callMsg.From.String()))
	signer := types.NewEIP155Signer(big.NewInt(int64(decoder.wm.Config.ChainID)))
	gasLimit := fee.GasLimit.Uint64()

	//构建合约交易
	tx := types.NewTransaction(nonce, callMsg.To,
		amount, gasLimit, fee.GasPrice, data)

	rawHex, err := rlp.EncodeToBytes(tx)
	if err != nil {
		decoder.wm.Log.Error("Transaction RLP encode failed, err:", err)
		return openwallet.Errorf(openwallet.ErrCreateRawSmartContractTransactionFailed, err.Error())
	}

	msg := signer.Hash(tx)

	if rawTx.Signatures == nil {
		rawTx.Signatures = make(map[string][]*openwallet.KeySignature)
	}

	signature := openwallet.KeySignature{
		EccType: decoder.wm.Config.CurveType,
		Nonce:   "0x" + strconv.FormatUint(nonce, 16),
		Address: addr,
		Message: hex.EncodeToString(msg[:]),
		RSV:     true,
	}
	keySignList = append(keySignList, &signature)

	rawTx.Raw = hex.EncodeToString(rawHex)
	rawTx.RawType = openwallet.TxRawTypeHex
	rawTx.Signatures[rawTx.Account.AccountID] = keySignList
	rawTx.FeeRate = gasprice.String()
	rawTx.Fees = totalFeeDecimal.String()
	rawTx.TxFrom = strings.ToLower(callMsg.From.String())
	rawTx.TxTo = strings.ToLower(callMsg.To.String())
	rawTx.IsBuilt = true

	return nil
}

// SubmitRawTransaction 广播交易单
func (decoder *EthContractDecoder) SubmitSmartContractRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.SmartContractRawTransaction) (*openwallet.SmartContractReceipt, *openwallet.Error) {

	err := decoder.VerifyRawTransaction(wrapper, rawTx)

	from := rawTx.TxFrom
	sig := rawTx.Signatures[rawTx.Account.AccountID][0].Signature

	signer := types.NewEIP155Signer(big.NewInt(int64(decoder.wm.Config.ChainID)))
	tx := &types.Transaction{}
	var decodeErr error
	//解析原始交易单
	switch rawTx.RawType {
	case openwallet.TxRawTypeHex:
		rawBytes := hexutil.MustDecode(AppendOxToAddress(rawTx.Raw))
		decodeErr = rlp.DecodeBytes(rawBytes, tx)
	case openwallet.TxRawTypeJSON:
		decodeErr = tx.UnmarshalJSON([]byte(rawTx.Raw))
	case openwallet.TxRawTypeBase64:
		rawBytes, _ := base64.StdEncoding.DecodeString(rawTx.Raw)
		decodeErr = rlp.DecodeBytes(rawBytes, tx)
	}

	if decodeErr != nil {
		decoder.wm.Log.Error("transaction RLP decode failed, err:", decodeErr)
		return nil, openwallet.Errorf(openwallet.ErrSubmitRawSmartContractTransactionFailed, decodeErr.Error())
	}

	tx, err = tx.WithSignature(signer, ethcom.FromHex(sig))
	if err != nil {
		decoder.wm.Log.Std.Error("tx with signature failed, err=%v ", err)
		return nil, openwallet.Errorf(openwallet.ErrSubmitRawSmartContractTransactionFailed, "tx with signature failed. ")
	}

	rawTxPara, err := rlp.EncodeToBytes(tx)
	if err != nil {
		decoder.wm.Log.Std.Error("encode tx to rlp failed, err=%v ", err)
		return nil, openwallet.Errorf(openwallet.ErrSubmitRawSmartContractTransactionFailed, "encode tx to rlp failed. ")
	}

	txid, err := decoder.wm.SendRawTransaction(hexutil.Encode(rawTxPara))
	if err != nil {
		decoder.wm.Log.Std.Error("sent raw tx faild, err=%v", err)
		//交易失败重置地址nonce
		decoder.wm.UpdateAddressNonce(wrapper, from, 0)
		return nil, openwallet.Errorf(openwallet.ErrSubmitRawSmartContractTransactionFailed, "sent raw tx faild. unexpected error: %v", err)
	}

	//交易成功，地址nonce+1并记录到缓存
	decoder.wm.UpdateAddressNonce(wrapper, from, tx.Nonce()+1)

	rawTx.TxID = txid
	rawTx.IsSubmit = true

	owtx := &openwallet.SmartContractReceipt{
		Coin:  rawTx.Coin,
		TxID:  rawTx.TxID,
		From:  rawTx.TxFrom,
		To:    rawTx.TxTo,
		Value: rawTx.Value,
		Fees:  rawTx.Fees,
	}

	owtx.GenWxID()

	decoder.wm.Log.Infof("rawTx.AwaitResult = %v", rawTx.AwaitResult)
	//等待出块结果返回交易回执
	if rawTx.AwaitResult {
		bs := decoder.wm.GetBlockScanner()
		if bs == nil {
			decoder.wm.Log.Errorf("adapter blockscanner is nil")
			return owtx, nil
		}

		addrs := make(map[string]openwallet.ScanTargetResult)
		contract := &rawTx.Coin.Contract
		if contract == nil {
			decoder.wm.Log.Errorf("rawTx.Coin.Contract is nil")
			return owtx, nil
		}

		addrs[contract.Address] = openwallet.ScanTargetResult{SourceKey: contract.ContractID, Exist: true, TargetInfo: contract}

		scanTargetFunc := func(target openwallet.ScanTargetParam) openwallet.ScanTargetResult {
			result := addrs[target.ScanTarget]
			if result.Exist {
				return result
			}
			return openwallet.ScanTargetResult{SourceKey: "", Exist: false, TargetInfo: nil}
		}

		//默认超时90秒
		if rawTx.AwaitTimeout == 0 {
			rawTx.AwaitTimeout = 90
		}

		sleepSecond := 2 * time.Second

		//计算过期时间
		currentServerTime := time.Now()
		expiredTime := currentServerTime.Add(time.Duration(rawTx.AwaitTimeout) * time.Second)

		//等待交易单报块结果
		for {

			//当前重试时间
			currentReDoTime := time.Now()

			//decoder.wm.Log.Debugf("currentReDoTime = %s", currentReDoTime.String())
			//decoder.wm.Log.Debugf("expiredTime = %s", expiredTime.String())

			//超时终止
			if currentReDoTime.Unix() >= expiredTime.Unix() {
				break
			}

			_, contractResult, extractErr := bs.ExtractTransactionAndReceiptData(owtx.TxID, scanTargetFunc)
			if extractErr != nil {
				decoder.wm.Log.Errorf("ExtractTransactionAndReceiptData failed, err: %v", extractErr)
				return owtx, nil
			}

			//tx := txResult[contract.ContractID]
			receipt := contractResult[contract.ContractID]

			if receipt != nil {
				return receipt, nil
			}

			//等待sleepSecond秒重试
			time.Sleep(sleepSecond)
		}

	}

	return owtx, nil
}

// VerifyRawTransaction 验证交易单，验证交易单并返回加入签名后的交易单
func (decoder *EthContractDecoder) VerifyRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.SmartContractRawTransaction) error {

	if rawTx.Signatures == nil || len(rawTx.Signatures) == 0 {
		//decoder.wm.Log.Std.Error("len of signatures error. ")
		return openwallet.Errorf(openwallet.ErrVerifyRawTransactionFailed, "transaction signature is empty")
	}

	accountSig, exist := rawTx.Signatures[rawTx.Account.AccountID]
	if !exist {
		decoder.wm.Log.Std.Error("wallet[%v] signature not found ", rawTx.Account.AccountID)
		return errors.New("wallet signature not found ")
	}

	if len(accountSig) == 0 {
		//decoder.wm.Log.Std.Error("len of signatures error. ")
		return openwallet.Errorf(openwallet.ErrVerifyRawTransactionFailed, "transaction signature is empty")
	}

	sig := accountSig[0].Signature
	msg := accountSig[0].Message
	pubkey := accountSig[0].Address.PublicKey
	//curveType := rawTx.Signatures[rawTx.Account.AccountID][0].EccType

	//decoder.wm.Log.Debug("-- pubkey:", pubkey)
	//decoder.wm.Log.Debug("-- message:", msg)
	//decoder.wm.Log.Debug("-- Signature:", sig)
	signature := ethcom.FromHex(sig)
	if signature == nil || len(signature) == 0 {
		return openwallet.Errorf(openwallet.ErrVerifyRawTransactionFailed, "transaction signature is empty")
	}
	publickKey := owcrypt.PointDecompress(ethcom.FromHex(pubkey), owcrypt.ECC_CURVE_SECP256K1)
	publickKey = publickKey[1:len(publickKey)]
	ret := owcrypt.Verify(publickKey, nil, ethcom.FromHex(msg), signature[0:len(signature)-1], owcrypt.ECC_CURVE_SECP256K1)
	if ret != owcrypt.SUCCESS {
		return fmt.Errorf("verify error, ret:%v\n", "0x"+strconv.FormatUint(uint64(ret), 16))
	}

	return nil
}

// GetTokenMetadata 根据合约地址查询token元数据
func (decoder *EthContractDecoder) GetTokenMetadata(contract string) (*openwallet.SmartContract, error) {

	var (
		tokenName     string
		tokenSymbol   string
		tokenDecimals uint8
		out           []interface{}
		contractInfo  *openwallet.SmartContract
	)

	contractAddress := decoder.wm.CustomAddressEncodeFunc(contract)
	contractId := openwallet.GenContractID(decoder.wm.Symbol(), contractAddress)

	if decoder.wm.Config.UseQNSingleFlightRPC == 1 {

		params := []interface{}{
			map[string]interface{}{
				"contract": contract,
			},
		}
		result, err := decoder.wm.WalletClient.Call("qn_getTokenMetadataByContractAddress", params)
		if err == nil {
			tokenName = result.Get("name").String()
			tokenSymbol = result.Get("symbol").String()
			tokenDecimals = uint8(result.Get("decimals").Uint())
		}

	} else {
		//erc20
		bc := bind.NewBoundContract(ethcom.HexToAddress(contractAddress), ERC20_ABI, decoder.wm.RawClient, decoder.wm.RawClient, nil)
		bc.Call(&bind.CallOpts{}, &out, "name")
		if out != nil && len(out) > 0 {
			tokenName = common.NewString(out[0]).String()
			out = nil
		}

		bc.Call(&bind.CallOpts{}, &out, "symbol")
		if out != nil && len(out) > 0 {
			tokenSymbol = common.NewString(out[0]).String()
			out = nil
		}

		bc.Call(&bind.CallOpts{}, &out, "decimals")
		if out != nil && len(out) > 0 {
			tokenDecimals = common.NewString(out[0]).UInt8()
			out = nil
		}

	}

	contractInfo = &openwallet.SmartContract{
		ContractID: contractId,
		Symbol:     decoder.wm.Symbol(),
		Address:    contractAddress,
		Token:      tokenSymbol,
		Protocol:   openwallet.InterfaceTypeERC20,
		Name:       tokenName,
		Decimals:   uint64(tokenDecimals),
	}

	return contractInfo, nil
}
