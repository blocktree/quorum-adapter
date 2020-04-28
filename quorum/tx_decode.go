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
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/blocktree/openwallet/v2/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/shopspring/decimal"
	"math/big"
	"sort"
	"strconv"
	"time"

	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/v2/openwallet"
	ethcom "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

type EthTransactionDecoder struct {
	openwallet.TransactionDecoderBase
	wm *WalletManager //钱包管理者
}

//NewTransactionDecoder 交易单解析器
func NewTransactionDecoder(wm *WalletManager) *EthTransactionDecoder {
	decoder := EthTransactionDecoder{}
	decoder.wm = wm
	return &decoder
}

func (decoder *EthTransactionDecoder) GetRawTransactionFeeRate() (feeRate string, unit string, err error) {
	price, err := decoder.wm.GetGasPrice()
	if err != nil {
		decoder.wm.Log.Errorf("get gas price failed, err=%v", err)
		return "", "Gas", err
	}

	pricedecimal := common.BigIntToDecimals(price, decoder.wm.Decimal())
	return pricedecimal.String(), "Gas", nil
}

func (decoder *EthTransactionDecoder) CreateSimpleRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction, tmpNonce *uint64) error {

	var (
		accountID       = rawTx.Account.AccountID
		findAddrBalance *AddrBalance
		feeInfo         *txFeeInfo
	)

	//获取wallet
	addresses, err := wrapper.GetAddressList(0, -1,
		"AccountID", accountID)
	if err != nil {
		return openwallet.NewError(openwallet.ErrAddressNotFound, err.Error())
	}

	if len(addresses) == 0 {
		return openwallet.Errorf(openwallet.ErrAccountNotAddress, "[%s] have not addresses", accountID)
	}

	searchAddrs := make([]string, 0)
	for _, address := range addresses {
		searchAddrs = append(searchAddrs, address.Address)
	}

	addrBalanceArray, err := decoder.wm.Blockscanner.GetBalanceByAddress(searchAddrs...)
	if err != nil {
		return openwallet.NewError(openwallet.ErrCallFullNodeAPIFailed, err.Error())
	}

	var amountStr, to string
	for k, v := range rawTx.To {
		to = k
		amountStr = v
		break
	}

	amount := common.StringNumToBigIntWithExp(amountStr, decoder.wm.Decimal())

	//地址余额从大到小排序
	sort.Slice(addrBalanceArray, func(i int, j int) bool {
		a_amount, _ := decimal.NewFromString(addrBalanceArray[i].Balance)
		b_amount, _ := decimal.NewFromString(addrBalanceArray[j].Balance)
		if a_amount.LessThan(b_amount) {
			return true
		} else {
			return false
		}
	})

	for _, addrBalance := range addrBalanceArray {

		//检查余额是否超过最低转账
		addrBalance_BI := common.StringNumToBigIntWithExp(addrBalance.Balance, decoder.wm.Decimal())

		//计算手续费
		feeInfo, err = decoder.wm.GetTransactionFeeEstimated(addrBalance.Address, to, amount, nil)
		if err != nil {
			//decoder.wm.Log.Std.Error("GetTransactionFeeEstimated from[%v] -> to[%v] failed, err=%v", addrBalance.Address, to, err)
			continue
		}

		if rawTx.FeeRate != "" {
			feeInfo.GasPrice = common.StringNumToBigIntWithExp(rawTx.FeeRate, decoder.wm.Decimal())
			feeInfo.CalcFee()
		}

		//总消耗数量 = 转账数量 + 手续费
		totalAmount := new(big.Int)
		totalAmount.Add(amount, feeInfo.Fee)

		if addrBalance_BI.Cmp(totalAmount) < 0 {
			continue
		}

		//只要找到一个合适使用的地址余额就停止遍历
		findAddrBalance = &AddrBalance{Address: addrBalance.Address, Balance: addrBalance_BI}
		break
	}

	if findAddrBalance == nil {
		return openwallet.Errorf(openwallet.ErrInsufficientBalanceOfAccount, "the balance: %s is not enough", amountStr)
	}

	//最后创建交易单
	createTxErr := decoder.createRawTransaction(
		wrapper,
		rawTx,
		findAddrBalance,
		feeInfo,
		"",
		tmpNonce)
	if createTxErr != nil {
		return createTxErr
	}

	return nil
}

func (decoder *EthTransactionDecoder) CreateErc20TokenRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	var (
		accountID       = rawTx.Account.AccountID
		findAddrBalance *AddrBalance
		feeInfo         *txFeeInfo
		errBalance      string
		errTokenBalance string
		callData        string
	)

	tokenDecimals := int32(rawTx.Coin.Contract.Decimals)
	contractAddress := rawTx.Coin.Contract.Address

	//获取wallet
	addresses, err := wrapper.GetAddressList(0, -1,
		"AccountID", accountID)
	if err != nil {
		return openwallet.NewError(openwallet.ErrAddressNotFound, err.Error())
	}

	if len(addresses) == 0 {
		return openwallet.Errorf(openwallet.ErrAccountNotAddress, "[%s] have not addresses", accountID)
	}

	searchAddrs := make([]string, 0)
	for _, address := range addresses {
		searchAddrs = append(searchAddrs, address.Address)
	}

	addrBalanceArray, err := decoder.wm.ContractDecoder.GetTokenBalanceByAddress(rawTx.Coin.Contract, searchAddrs...)
	if err != nil {
		return openwallet.NewError(openwallet.ErrCallFullNodeAPIFailed, err.Error())
	}

	var amountStr, to string
	for k, v := range rawTx.To {
		to = k
		amountStr = v
		break
	}

	//地址余额从大到小排序
	sort.Slice(addrBalanceArray, func(i int, j int) bool {
		a_amount, _ := decimal.NewFromString(addrBalanceArray[i].Balance.Balance)
		b_amount, _ := decimal.NewFromString(addrBalanceArray[j].Balance.Balance)
		if a_amount.LessThan(b_amount) {
			return true
		} else {
			return false
		}
	})

	tokenBalanceNotEnough := false
	balanceNotEnough := false

	for _, addrBalance := range addrBalanceArray {
		callData = ""

		//检查余额是否超过最低转账
		addrBalance_BI := common.StringNumToBigIntWithExp(addrBalance.Balance.Balance, tokenDecimals)

		amount := common.StringNumToBigIntWithExp(amountStr, tokenDecimals)

		if addrBalance_BI.Cmp(amount) < 0 {
			errTokenBalance = fmt.Sprintf("the token balance of all addresses is not enough")
			tokenBalanceNotEnough = true
			continue
		}

		data, createErr := decoder.wm.EncodeABIParam(ERC20_ABI, "transfer", decoder.wm.CustomAddressDecodeFunc(to), amount.String())
		if createErr != nil {
			continue
		}

		//decoder.wm.Log.Debug("sumAmount:", sumAmount)
		//计算手续费
		fee, createErr := decoder.wm.GetTransactionFeeEstimated(addrBalance.Balance.Address, contractAddress, nil, data)
		if createErr != nil {
			//decoder.wm.Log.Std.Error("GetTransactionFeeEstimated from[%v] -> to[%v] failed, err=%v", addrBalance.Balance.Address, to, createErr)
			return createErr
		}

		if rawTx.FeeRate != "" {
			fee.GasPrice = common.StringNumToBigIntWithExp(rawTx.FeeRate, decoder.wm.Decimal()) //ConvertToBigInt(rawTx.FeeRate, 16)
			fee.CalcFee()
		}

		coinBalance, err := decoder.wm.GetAddrBalance(addrBalance.Balance.Address, "pending")
		if err != nil {
			continue
		}

		if coinBalance.Cmp(fee.Fee) < 0 {
			coinBalance := common.BigIntToDecimals(coinBalance, decoder.wm.Decimal())
			errBalance = fmt.Sprintf("the [%s] balance: %s is not enough to call smart contract", rawTx.Coin.Symbol, coinBalance.String())
			balanceNotEnough = true
			continue
		}

		//只要找到一个合适使用的地址余额就停止遍历
		findAddrBalance = &AddrBalance{Address: addrBalance.Balance.Address, Balance: coinBalance, TokenBalance: addrBalance_BI}
		feeInfo = fee
		callData = hex.EncodeToString(data)
		break
	}

	if findAddrBalance == nil {
		if tokenBalanceNotEnough {
			return openwallet.Errorf(openwallet.ErrInsufficientTokenBalanceOfAddress, errTokenBalance)
		}
		if balanceNotEnough {
			return openwallet.Errorf(openwallet.ErrInsufficientFees, errBalance)
		}
	}

	//最后创建交易单
	createTxErr := decoder.createRawTransaction(
		wrapper,
		rawTx,
		findAddrBalance,
		feeInfo,
		callData,
		nil)
	if createTxErr != nil {
		return createTxErr
	}

	return nil
}

//CreateRawTransaction 创建交易单
func (decoder *EthTransactionDecoder) CreateRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {
	if !rawTx.Coin.IsContract {
		return decoder.CreateSimpleRawTransaction(wrapper, rawTx, nil)
	}
	return decoder.CreateErc20TokenRawTransaction(wrapper, rawTx)
}

//SignRawTransaction 签名交易单
func (decoder *EthTransactionDecoder) SignRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	if rawTx.Signatures == nil || len(rawTx.Signatures) == 0 {
		//decoder.wm.Log.Std.Error("len of signatures error. ")
		return openwallet.Errorf(openwallet.ErrSignRawTransactionFailed, "transaction signature is empty")
	}

	key, err := wrapper.HDKey()
	if err != nil {
		decoder.wm.Log.Error("get HDKey from wallet wrapper failed, err=%v", err)
		return err
	}

	if _, exist := rawTx.Signatures[rawTx.Account.AccountID]; !exist {
		decoder.wm.Log.Std.Error("wallet[%v] signature not found ", rawTx.Account.AccountID)
		return openwallet.Errorf(openwallet.ErrSignRawTransactionFailed, "wallet signature not found ")
	}

	if len(rawTx.Signatures[rawTx.Account.AccountID]) != 1 {
		decoder.wm.Log.Error("signature failed in account[%v].", rawTx.Account.AccountID)
		return openwallet.Errorf(openwallet.ErrSignRawTransactionFailed, "signature failed in account.")
	}

	signnode := rawTx.Signatures[rawTx.Account.AccountID][0]
	fromAddr := signnode.Address

	childKey, _ := key.DerivedKeyWithPath(fromAddr.HDPath, owcrypt.ECC_CURVE_SECP256K1)
	keyBytes, err := childKey.GetPrivateKeyBytes()
	if err != nil {
		//decoder.wm.Log.Error("get private key bytes, err=", err)
		return openwallet.NewError(openwallet.ErrSignRawTransactionFailed, err.Error())
	}
	//prikeyStr := common.ToHex(keyBytes)
	//decoder.wm.Log.Debugf("pri:%v", common.ToHex(keyBytes))

	message, err := hex.DecodeString(signnode.Message)
	if err != nil {
		return err
	}

	signature, v, sigErr := owcrypt.Signature(keyBytes, nil, message, decoder.wm.CurveType())
	if sigErr != owcrypt.SUCCESS {
		return fmt.Errorf("transaction hash sign failed")
	}
	signature = append(signature, v)

	signnode.Signature = hex.EncodeToString(signature)

	//decoder.wm.Log.Debug("** pri:", hex.EncodeToString(keyBytes))
	//decoder.wm.Log.Debug("** message:", signnode.Message)
	//decoder.wm.Log.Debug("** Signature:", signnode.Signature)

	return nil
}

// SubmitRawTransaction 广播交易单
func (decoder *EthTransactionDecoder) SubmitRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) (*openwallet.Transaction, error) {

	if len(rawTx.Signatures) != 1 {
		decoder.wm.Log.Std.Error("len of signatures error. ")
		return nil, openwallet.Errorf(openwallet.ErrSubmitRawTransactionFailed, "len of signatures error. ")
	}

	if _, exist := rawTx.Signatures[rawTx.Account.AccountID]; !exist {
		decoder.wm.Log.Std.Error("wallet[%v] signature not found ", rawTx.Account.AccountID)
		return nil, openwallet.Errorf(openwallet.ErrSubmitRawTransactionFailed, "wallet signature not found ")
	}

	from := rawTx.Signatures[rawTx.Account.AccountID][0].Address.Address
	sig := rawTx.Signatures[rawTx.Account.AccountID][0].Signature

	//decoder.wm.Log.Debug("rawTx.ExtParam:", rawTx.ExtParam)

	signer := types.NewEIP155Signer(big.NewInt(int64(decoder.wm.Config.ChainID)))

	rawHex, err := hex.DecodeString(rawTx.RawHex)
	if err != nil {
		decoder.wm.Log.Error("rawTx.RawHex decode failed, err:", err)
		return nil, err
	}

	tx := &types.Transaction{}
	err = rlp.DecodeBytes(rawHex, tx)
	if err != nil {
		decoder.wm.Log.Error("transaction RLP decode failed, err:", err)
		return nil, err
	}

	//tx := types.NewTransaction(nonceSigned, ethcom.HexToAddress(to),
	//	amount, gaslimit.Uint64(), gasPrice, nil)
	tx, err = tx.WithSignature(signer, ethcom.FromHex(sig))
	if err != nil {
		decoder.wm.Log.Std.Error("tx with signature failed, err=%v ", err)
		return nil, openwallet.Errorf(openwallet.ErrSubmitRawTransactionFailed, "tx with signature failed. ")
	}

	//txstr, _ := json.MarshalIndent(tx, "", " ")
	//decoder.wm.Log.Debug("**after signed txStr:", string(txstr))

	rawTxPara, err := rlp.EncodeToBytes(tx)
	if err != nil {
		decoder.wm.Log.Std.Error("encode tx to rlp failed, err=%v ", err)
		return nil, openwallet.Errorf(openwallet.ErrSubmitRawTransactionFailed, "encode tx to rlp failed. ")
	}

	txid, err := decoder.wm.SendRawTransaction(hexutil.Encode(rawTxPara))
	if err != nil {
		decoder.wm.Log.Std.Error("sent raw tx faild, err=%v", err)
		//交易失败重置地址nonce
		decoder.wm.UpdateAddressNonce(wrapper, from, 0)
		return nil, openwallet.Errorf(openwallet.ErrSubmitRawTransactionFailed, "sent raw tx faild. unexpected error: %v", err)
	}

	//交易成功，地址nonce+1并记录到缓存
	decoder.wm.UpdateAddressNonce(wrapper, from, tx.Nonce()+1)

	rawTx.TxID = txid
	rawTx.IsSubmit = true

	//decoder.wm.Log.Debug("transaction[", txid, "] has been sent out.")

	decimals := int32(0)
	fees := "0"
	if rawTx.Coin.IsContract {
		decimals = int32(rawTx.Coin.Contract.Decimals)
		fees = "0"
	} else {
		decimals = int32(decoder.wm.Decimal())
		fees = rawTx.Fees
	}

	//记录一个交易单
	owtx := &openwallet.Transaction{
		From:       rawTx.TxFrom,
		To:         rawTx.TxTo,
		Amount:     rawTx.TxAmount,
		Coin:       rawTx.Coin,
		TxID:       rawTx.TxID,
		Decimal:    decimals,
		AccountID:  rawTx.Account.AccountID,
		Fees:       fees,
		SubmitTime: time.Now().Unix(),
		TxType:     0,
	}

	owtx.WxID = openwallet.GenTransactionWxID(owtx)

	return owtx, nil
}

//VerifyRawTransaction 验证交易单，验证交易单并返回加入签名后的交易单
func (decoder *EthTransactionDecoder) VerifyRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

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

	decoder.wm.Log.Debug("-- pubkey:", pubkey)
	decoder.wm.Log.Debug("-- message:", msg)
	decoder.wm.Log.Debug("-- Signature:", sig)
	signature := ethcom.FromHex(sig)
	publickKey := owcrypt.PointDecompress(ethcom.FromHex(pubkey), owcrypt.ECC_CURVE_SECP256K1)
	publickKey = publickKey[1:len(publickKey)]
	ret := owcrypt.Verify(publickKey, nil, ethcom.FromHex(msg), signature[0:len(signature)-1], owcrypt.ECC_CURVE_SECP256K1)
	if ret != owcrypt.SUCCESS {
		errinfo := fmt.Sprintf("verify error, ret:%v\n", "0x"+strconv.FormatUint(uint64(ret), 16))
		//fmt.Println(errinfo)
		return errors.New(errinfo)
	}

	return nil
}

//CreateSummaryRawTransaction 创建汇总交易，返回原始交易单数组
func (decoder *EthTransactionDecoder) CreateSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransaction, error) {
	var (
		rawTxWithErrArray []*openwallet.RawTransactionWithError
		rawTxArray        = make([]*openwallet.RawTransaction, 0)
		err               error
	)
	if sumRawTx.Coin.IsContract {
		rawTxWithErrArray, err = decoder.CreateErc20TokenSummaryRawTransaction(wrapper, sumRawTx)
	} else {
		rawTxWithErrArray, err = decoder.CreateSimpleSummaryRawTransaction(wrapper, sumRawTx)
	}
	if err != nil {
		return nil, err
	}
	for _, rawTxWithErr := range rawTxWithErrArray {
		if rawTxWithErr.Error != nil {
			continue
		}
		rawTxArray = append(rawTxArray, rawTxWithErr.RawTx)
	}
	return rawTxArray, nil
}

//CreateSimpleSummaryRawTransaction 创建QUORUM汇总交易
func (decoder *EthTransactionDecoder) CreateSimpleSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransactionWithError, error) {

	var (
		rawTxArray      = make([]*openwallet.RawTransactionWithError, 0)
		accountID       = sumRawTx.Account.AccountID
		minTransfer     = common.StringNumToBigIntWithExp(sumRawTx.MinTransfer, decoder.wm.Decimal())
		retainedBalance = common.StringNumToBigIntWithExp(sumRawTx.RetainedBalance, decoder.wm.Decimal())
	)

	if minTransfer.Cmp(retainedBalance) < 0 {
		return nil, openwallet.Errorf(openwallet.ErrCreateRawTransactionFailed, "mini transfer amount must be greater than address retained balance")
	}

	//获取wallet
	addresses, err := wrapper.GetAddressList(sumRawTx.AddressStartIndex, sumRawTx.AddressLimit,
		"AccountID", sumRawTx.Account.AccountID)
	if err != nil {
		return nil, err
	}

	if len(addresses) == 0 {
		return nil, openwallet.Errorf(openwallet.ErrAccountNotAddress, "[%s] have not addresses", accountID)
	}

	searchAddrs := make([]string, 0)
	for _, address := range addresses {
		searchAddrs = append(searchAddrs, address.Address)
	}

	addrBalanceArray, err := decoder.wm.Blockscanner.GetBalanceByAddress(searchAddrs...)
	if err != nil {
		return nil, err
	}

	for _, addrBalance := range addrBalanceArray {

		//检查余额是否超过最低转账
		addrBalance_BI := common.StringNumToBigIntWithExp(addrBalance.Balance, decoder.wm.Decimal())

		if addrBalance_BI.Cmp(minTransfer) < 0 {
			continue
		}
		//计算汇总数量 = 余额 - 保留余额
		sumAmount_BI := new(big.Int)
		sumAmount_BI.Sub(addrBalance_BI, retainedBalance)

		//decoder.wm.Log.Debug("sumAmount:", sumAmount)
		//计算手续费
		fee, createErr := decoder.wm.GetTransactionFeeEstimated(addrBalance.Address, sumRawTx.SummaryAddress, sumAmount_BI, nil)
		if createErr != nil {
			//decoder.wm.Log.Std.Error("GetTransactionFeeEstimated from[%v] -> to[%v] failed, err=%v", addrBalance.Address, sumRawTx.SummaryAddress, createErr)
			return nil, createErr
		}

		if sumRawTx.FeeRate != "" {
			fee.GasPrice = common.StringNumToBigIntWithExp(sumRawTx.FeeRate, decoder.wm.Decimal()) //ConvertToBigInt(rawTx.FeeRate, 16)
			if createErr != nil {
				decoder.wm.Log.Std.Error("fee rate passed through error, err=%v", createErr)
				return nil, createErr
			}
			fee.CalcFee()
		}

		//减去手续费
		sumAmount_BI.Sub(sumAmount_BI, fee.Fee)
		if sumAmount_BI.Cmp(big.NewInt(0)) <= 0 {
			continue
		}

		sumAmount := common.BigIntToDecimals(sumAmount_BI, decoder.wm.Decimal())
		fees := common.BigIntToDecimals(fee.Fee, decoder.wm.Decimal())

		decoder.wm.Log.Debugf("balance: %v", addrBalance.Balance)
		decoder.wm.Log.Debugf("fees: %v", fees)
		decoder.wm.Log.Debugf("sumAmount: %v", sumAmount)

		//创建一笔交易单
		rawTx := &openwallet.RawTransaction{
			Coin:    sumRawTx.Coin,
			Account: sumRawTx.Account,
			To: map[string]string{
				sumRawTx.SummaryAddress: sumAmount.StringFixed(decoder.wm.Decimal()),
			},
			Required: 1,
		}

		createTxErr := decoder.createRawTransaction(
			wrapper,
			rawTx,
			&AddrBalance{Address: addrBalance.Address, Balance: addrBalance_BI},
			fee,
			"",
			nil)
		rawTxWithErr := &openwallet.RawTransactionWithError{
			RawTx: rawTx,
			Error: createTxErr,
		}

		//创建成功，添加到队列
		rawTxArray = append(rawTxArray, rawTxWithErr)

	}

	return rawTxArray, nil
}

//CreateErc20TokenSummaryRawTransaction 创建ERC20Token汇总交易
func (decoder *EthTransactionDecoder) CreateErc20TokenSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransactionWithError, error) {

	var (
		rawTxArray         = make([]*openwallet.RawTransactionWithError, 0)
		accountID          = sumRawTx.Account.AccountID
		minTransfer        *big.Int
		retainedBalance    *big.Int
		feesSupportAccount *openwallet.AssetsAccount
		tmpNonce           uint64
	)

	// 如果有提供手续费账户，检查账户是否存在
	if feesAcount := sumRawTx.FeesSupportAccount; feesAcount != nil {
		account, supportErr := wrapper.GetAssetsAccountInfo(feesAcount.AccountID)
		if supportErr != nil {
			return nil, openwallet.Errorf(openwallet.ErrAccountNotFound, "can not find fees support account")
		}

		feesSupportAccount = account

		//获取手续费支持账户的地址nonce
		feesAddresses, feesSupportErr := wrapper.GetAddressList(0, 1,
			"AccountID", feesSupportAccount.AccountID)
		if feesSupportErr != nil {
			return nil, openwallet.NewError(openwallet.ErrAddressNotFound, "fees support account have not addresses")
		}

		if len(feesAddresses) == 0 {
			return nil, openwallet.Errorf(openwallet.ErrAccountNotAddress, "fees support account have not addresses")
		}

		nonce, feesSupportErr := decoder.wm.GetTransactionCount(feesAddresses[0].Address)
		if feesSupportErr != nil {
			return nil, openwallet.NewError(openwallet.ErrNonceInvaild, "fees support account get nonce failed")
		}
		tmpNonce = nonce
	}
	//tokenCoin := sumRawTx.Coin.Contract.Token
	tokenDecimals := int32(sumRawTx.Coin.Contract.Decimals)
	contractAddress := sumRawTx.Coin.Contract.Address
	//coinDecimals := decoder.wm.Decimal()

	minTransfer = common.StringNumToBigIntWithExp(sumRawTx.MinTransfer, tokenDecimals)
	retainedBalance = common.StringNumToBigIntWithExp(sumRawTx.RetainedBalance, tokenDecimals)

	if minTransfer.Cmp(retainedBalance) < 0 {
		return nil, openwallet.Errorf(openwallet.ErrCreateRawTransactionFailed, "mini transfer amount must be greater than address retained balance")
	}

	//获取wallet
	addresses, err := wrapper.GetAddressList(sumRawTx.AddressStartIndex, sumRawTx.AddressLimit,
		"AccountID", sumRawTx.Account.AccountID)
	if err != nil {
		return nil, err
	}

	if len(addresses) == 0 {
		return nil, openwallet.Errorf(openwallet.ErrAccountNotAddress, "[%s] have not addresses", accountID)
	}

	searchAddrs := make([]string, 0)
	for _, address := range addresses {
		searchAddrs = append(searchAddrs, address.Address)
	}

	//查询Token余额
	addrBalanceArray, err := decoder.wm.ContractDecoder.GetTokenBalanceByAddress(sumRawTx.Coin.Contract, searchAddrs...)
	if err != nil {
		return nil, err
	}

	for _, addrBalance := range addrBalanceArray {

		//检查余额是否超过最低转账
		addrBalance_BI := common.StringNumToBigIntWithExp(addrBalance.Balance.Balance, tokenDecimals)

		if addrBalance_BI.Cmp(minTransfer) < 0 || addrBalance_BI.Cmp(big.NewInt(0)) <= 0 {
			continue
		}
		//计算汇总数量 = 余额 - 保留余额
		sumAmount_BI := new(big.Int)
		sumAmount_BI.Sub(addrBalance_BI, retainedBalance)

		callData, err := decoder.wm.EncodeABIParam(ERC20_ABI, "transfer", sumRawTx.SummaryAddress, sumAmount_BI.String())

		//decoder.wm.Log.Debug("sumAmount:", sumAmount)
		//计算手续费
		fee, createErr := decoder.wm.GetTransactionFeeEstimated(addrBalance.Balance.Address, contractAddress, nil, callData)
		if createErr != nil {
			//decoder.wm.Log.Std.Error("GetTransactionFeeEstimated from[%v] -> to[%v] failed, err=%v", addrBalance.Balance.Address, sumRawTx.SummaryAddress, createErr)
			return nil, createErr
		}

		if sumRawTx.FeeRate != "" {
			fee.GasPrice = common.StringNumToBigIntWithExp(sumRawTx.FeeRate, decoder.wm.Decimal()) //ConvertToBigInt(rawTx.FeeRate, 16)
			if createErr != nil {
				decoder.wm.Log.Std.Error("fee rate passed through error, err=%v", createErr)
				return nil, createErr
			}
			fee.CalcFee()
		}

		sumAmount := common.BigIntToDecimals(sumAmount_BI, tokenDecimals)
		fees := common.BigIntToDecimals(fee.Fee, decoder.wm.Decimal())

		coinBalance, createErr := decoder.wm.GetAddrBalance(addrBalance.Balance.Address, "pending")
		if err != nil {
			continue
		}

		//判断主币余额是否够手续费
		if coinBalance.Cmp(fee.Fee) < 0 {

			//有手续费账户支持
			if feesSupportAccount != nil {

				//通过手续费账户创建交易单
				supportAddress := addrBalance.Balance.Address
				supportAmount := decimal.Zero
				feesSupportScale, _ := decimal.NewFromString(sumRawTx.FeesSupportAccount.FeesSupportScale)
				fixSupportAmount, _ := decimal.NewFromString(sumRawTx.FeesSupportAccount.FixSupportAmount)

				//优先采用固定支持数量
				if fixSupportAmount.GreaterThan(decimal.Zero) {
					supportAmount = fixSupportAmount
				} else {
					//没有固定支持数量，有手续费倍率，计算支持数量
					if feesSupportScale.GreaterThan(decimal.Zero) {
						supportAmount = feesSupportScale.Mul(fees)
					} else {
						//默认支持数量为手续费
						supportAmount = fees
					}
				}

				decoder.wm.Log.Debugf("create transaction for fees support account")
				decoder.wm.Log.Debugf("fees account: %s", feesSupportAccount.AccountID)
				decoder.wm.Log.Debugf("mini support amount: %s", fees.String())
				decoder.wm.Log.Debugf("allow support amount: %s", supportAmount.String())
				decoder.wm.Log.Debugf("support address: %s", supportAddress)

				supportCoin := openwallet.Coin{
					Symbol:     sumRawTx.Coin.Symbol,
					IsContract: false,
				}

				//创建一笔交易单
				rawTx := &openwallet.RawTransaction{
					Coin:    supportCoin,
					Account: feesSupportAccount,
					To: map[string]string{
						addrBalance.Balance.Address: supportAmount.String(),
					},
					Required: 1,
				}

				createTxErr := decoder.CreateSimpleRawTransaction(wrapper, rawTx, &tmpNonce)
				rawTxWithErr := &openwallet.RawTransactionWithError{
					RawTx: rawTx,
					Error: openwallet.ConvertError(createTxErr),
				}

				//创建成功，添加到队列
				rawTxArray = append(rawTxArray, rawTxWithErr)

				//需要手续费支持的地址会有很多个，nonce要连续递增以保证交易广播生效
				tmpNonce++

				//汇总下一个
				continue
			}
		}

		decoder.wm.Log.Debugf("balance: %v", addrBalance.Balance.Balance)
		decoder.wm.Log.Debugf("%s fees: %v", sumRawTx.Coin.Symbol, fees)
		decoder.wm.Log.Debugf("sumAmount: %v", sumAmount)

		//创建一笔交易单
		rawTx := &openwallet.RawTransaction{
			Coin:    sumRawTx.Coin,
			Account: sumRawTx.Account,
			To: map[string]string{
				sumRawTx.SummaryAddress: sumAmount.StringFixed(int32(tokenDecimals)),
			},
			Required: 1,
		}

		createTxErr := decoder.createRawTransaction(
			wrapper,
			rawTx,
			&AddrBalance{Address: addrBalance.Balance.Address, Balance: coinBalance, TokenBalance: addrBalance_BI},
			fee,
			hex.EncodeToString(callData),
			nil)
		rawTxWithErr := &openwallet.RawTransactionWithError{
			RawTx: rawTx,
			Error: createTxErr,
		}

		//创建成功，添加到队列
		rawTxArray = append(rawTxArray, rawTxWithErr)

	}

	return rawTxArray, nil
}

//createRawTransaction
func (decoder *EthTransactionDecoder) createRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction, addrBalance *AddrBalance, fee *txFeeInfo, callData string, tmpNonce *uint64) *openwallet.Error {

	var (
		accountTotalSent = decimal.Zero
		txFrom           = make([]string, 0)
		txTo             = make([]string, 0)
		keySignList      = make([]*openwallet.KeySignature, 0)
		amountStr        string
		destination      string
		tx               *types.Transaction
	)

	isContract := rawTx.Coin.IsContract
	//contractAddress := rawTx.Coin.Contract.Address
	//tokenCoin := rawTx.Coin.Contract.Token
	tokenDecimals := int32(rawTx.Coin.Contract.Decimals)
	//coinDecimals := decoder.wm.Decimal()

	for k, v := range rawTx.To {
		destination = k
		amountStr = v
		break
	}

	//计算账户的实际转账amount
	accountTotalSentAddresses, findErr := wrapper.GetAddressList(0, -1, "AccountID", rawTx.Account.AccountID, "Address", destination)
	if findErr != nil || len(accountTotalSentAddresses) == 0 {
		amountDec, _ := decimal.NewFromString(amountStr)
		accountTotalSent = accountTotalSent.Add(amountDec)
	}

	txFrom = []string{fmt.Sprintf("%s:%s", addrBalance.Address, amountStr)}
	txTo = []string{fmt.Sprintf("%s:%s", destination, amountStr)}

	gasprice := common.BigIntToDecimals(fee.GasPrice, decoder.wm.Decimal())
	totalFeeDecimal := common.BigIntToDecimals(fee.Fee, decoder.wm.Decimal())

	feesDec, _ := decimal.NewFromString(rawTx.Fees)
	accountTotalSent = accountTotalSent.Add(feesDec)
	accountTotalSent = decimal.Zero.Sub(accountTotalSent)

	rawTx.FeeRate = gasprice.String()
	rawTx.Fees = totalFeeDecimal.String()
	//rawTx.ExtParam = string(extparastr)
	rawTx.TxAmount = accountTotalSent.String()
	rawTx.TxFrom = txFrom
	rawTx.TxTo = txTo

	addr, err := wrapper.GetAddress(addrBalance.Address)
	if err != nil {
		return openwallet.NewError(openwallet.ErrAccountNotAddress, err.Error())
	}

	var nonce uint64
	if tmpNonce == nil {
		txNonce := decoder.wm.GetAddressNonce(wrapper, addrBalance.Address)
		//decoder.wm.Log.Debugf("txNonce: %d", txNonce)
		nonce = txNonce
	} else {
		nonce = *tmpNonce
	}

	//decoder.wm.Log.Debug("chainID:", decoder.wm.GetConfig().ChainID)
	signer := types.NewEIP155Signer(big.NewInt(int64(decoder.wm.Config.ChainID)))

	gasLimit := fee.GasLimit.Uint64()

	if isContract {
		//构建合约交易
		amount := common.StringNumToBigIntWithExp(amountStr, tokenDecimals)
		if addrBalance.TokenBalance.Cmp(amount) < 0 {
			return openwallet.Errorf(openwallet.ErrInsufficientTokenBalanceOfAddress, "the token balance: %s is not enough", amountStr)
			//return openwallet.Errorf("the token balance: %s is not enough", amountStr)
		}

		if addrBalance.Balance.Cmp(fee.Fee) < 0 {
			coinBalance := common.BigIntToDecimals(addrBalance.Balance, decoder.wm.Decimal())
			return openwallet.Errorf(openwallet.ErrInsufficientFees, "the [%s] balance: %s is not enough to call smart contract", rawTx.Coin.Symbol, coinBalance)
			//return openwallet.Errorf("the [%s] balance: %s is not enough to call smart contract", rawTx.Coin.Symbol, coinBalance)
		}

		tx = types.NewTransaction(nonce, ethcom.HexToAddress(decoder.wm.CustomAddressDecodeFunc(rawTx.Coin.Contract.Address)),
			big.NewInt(0), gasLimit, fee.GasPrice, ethcom.FromHex(callData))
	} else {
		//构建QUORUM交易
		amount := common.StringNumToBigIntWithExp(amountStr, decoder.wm.Decimal())

		totalAmount := new(big.Int)
		totalAmount.Add(amount, fee.Fee)
		if addrBalance.Balance.Cmp(totalAmount) < 0 {
			return openwallet.Errorf(openwallet.ErrInsufficientFees, "the [%s] balance: %s is not enough", rawTx.Coin.Symbol, amountStr)
			//return openwallet.Errorf("the [%s] balance: %s is not enough", rawTx.Coin.Symbol, amountStr)
		}

		tx = types.NewTransaction(nonce, ethcom.HexToAddress(decoder.wm.CustomAddressDecodeFunc(destination)),
			amount, gasLimit, fee.GasPrice, []byte(""))
	}

	rawHex, err := rlp.EncodeToBytes(tx)
	if err != nil {
		decoder.wm.Log.Error("Transaction RLP encode failed, err:", err)
		return openwallet.ConvertError(err)
	}

	//txstr, _ := json.MarshalIndent(tx, "", " ")
	//decoder.wm.Log.Debug("**txStr:", string(txstr))
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

	rawTx.RawHex = hex.EncodeToString(rawHex)
	rawTx.Signatures[rawTx.Account.AccountID] = keySignList
	rawTx.IsBuilt = true

	return nil
}

// CreateSummaryRawTransactionWithError 创建汇总交易，返回能原始交易单数组（包含带错误的原始交易单）
func (decoder *EthTransactionDecoder) CreateSummaryRawTransactionWithError(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransactionWithError, error) {
	if sumRawTx.Coin.IsContract {
		return decoder.CreateErc20TokenSummaryRawTransaction(wrapper, sumRawTx)
	} else {
		return decoder.CreateSimpleSummaryRawTransaction(wrapper, sumRawTx)
	}
}
