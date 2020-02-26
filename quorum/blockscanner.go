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

//block scan db内容:
//1. 扫描过的blockheader
//2. unscanned tx
//3. block height, block hash

import (
	"github.com/blocktree/openwallet/common"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"math/big"
	"strings"
	"time"

	"github.com/blocktree/openwallet/openwallet"

	//	"golang.org/x/text/currency"

	"fmt"
)

const (
	//BLOCK_CHAIN_BUCKET = "blockchain" //区块链数据集合
	//periodOfTask      = 5 * time.Second //定时任务执行隔间
	MAX_EXTRACTING_SIZE = 15 //并发的扫描线程数

)

type BlockScanner struct {
	*openwallet.BlockScannerBase
	CurrentBlockHeight   uint64         //当前区块高度
	extractingCH         chan struct{}  //扫描工作令牌
	wm                   *WalletManager //钱包管理者
	IsScanMemPool        bool           //是否扫描交易池
	RescanLastBlockCount uint64         //重扫上N个区块数量
}

//ExtractResult 扫描完成的提取结果
type ExtractResult struct {
	extractData         map[string][]*openwallet.TxExtractData
	extractContractData map[string]*openwallet.SmartContractReceipt
	TxID                string
	BlockHeight         uint64
	Success             bool
}

//SaveResult 保存结果
type SaveResult struct {
	TxID        string
	BlockHeight uint64
	Success     bool
}

//NewBTCBlockScanner 创建区块链扫描器
func NewBlockScanner(wm *WalletManager) *BlockScanner {
	bs := BlockScanner{
		BlockScannerBase: openwallet.NewBlockScannerBase(),
	}

	bs.extractingCH = make(chan struct{}, MAX_EXTRACTING_SIZE)
	bs.wm = wm
	bs.IsScanMemPool = false
	bs.RescanLastBlockCount = 0

	//设置扫描任务
	bs.SetTask(bs.ScanBlockTask)

	return &bs
}

//SetRescanBlockHeight 重置区块链扫描高度
func (bs *BlockScanner) SetRescanBlockHeight(height uint64) error {
	height = height - 1
	if height < 0 {
		return fmt.Errorf("block height to rescan must greater than 0 ")
	}

	block, err := bs.wm.GetBlockByNum(height, false)
	if err != nil {
		bs.wm.Log.Errorf("get block spec by block number[%v] failed, err=%v", height, err)
		return err
	}

	err = bs.SaveLocalBlockHead(height, block.BlockHash)
	if err != nil {
		bs.wm.Log.Errorf("save local block scanned failed, err=%v", err)
		return err
	}

	return nil
}

func (bs *BlockScanner) newBlockNotify(block *EthBlock, isFork bool) {
	header := block.CreateOpenWalletBlockHeader()
	header.Fork = isFork
	header.Symbol = bs.wm.Config.Symbol
	bs.NewBlockNotify(header)
}

func (bs *BlockScanner) ScanBlock(height uint64) error {
	curBlock, err := bs.wm.GetBlockByNum(height, true)
	if err != nil {
		bs.wm.Log.Errorf("EthGetBlockSpecByBlockNum failed, err = %v", err)
		return err
	}

	err = bs.BatchExtractTransaction(height, curBlock.Transactions)
	if err != nil {
		bs.wm.Log.Errorf("BatchExtractTransaction failed, err = %v", err)
		return err
	}

	bs.newBlockNotify(curBlock, false)

	return nil
}

//rescanFailedRecord 重扫失败记录
func (bs *BlockScanner) RescanFailedRecord() {

	var (
		blockMap = make(map[uint64][]string)
	)

	list, err := bs.GetUnscanRecords()
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get rescan data; unexpected error: %v", err)
	}

	//组合成批处理
	for _, r := range list {

		if _, exist := blockMap[r.BlockHeight]; !exist {
			blockMap[r.BlockHeight] = make([]string, 0)
		}

		if len(r.TxID) > 0 {
			arr := blockMap[r.BlockHeight]
			arr = append(arr, r.TxID)

			blockMap[r.BlockHeight] = arr
		}
	}

	for height, _ := range blockMap {

		if height == 0 {
			continue
		}

		bs.wm.Log.Std.Info("block scanner rescanning height: %d ...", height)

		block, err := bs.wm.GetBlockByNum(height, true)
		if err != nil {
			bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)
			continue
		}

		batchErr := bs.BatchExtractTransaction(block.BlockHeight, block.Transactions)
		if batchErr != nil {
			bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", batchErr)
			continue
		}

		//删除未扫记录
		bs.DeleteUnscanRecord(height)
	}
}

func (bs *BlockScanner) ScanBlockTask() {

	//获取本地区块高度
	blockHeader, err := bs.GetScannedBlockHeader()
	if err != nil {
		bs.wm.Log.Errorf("block scanner can not get new block height; unexpected error: %v", err)
		return
	}

	curBlockHeight := blockHeader.Height
	curBlockHash := blockHeader.Hash
	var previousHeight uint64 = 0
	for {

		if !bs.Scanning {
			//区块扫描器已暂停，马上结束本次任务
			return
		}

		maxBlockHeight, err := bs.wm.GetBlockNumber()
		if err != nil {
			bs.wm.Log.Errorf("get max height of eth failed, err=%v", err)
			break
		}

		bs.wm.Log.Info("current block height:", curBlockHeight, " maxBlockHeight:", maxBlockHeight)
		if curBlockHeight >= maxBlockHeight {
			bs.wm.Log.Infof("block scanner has done with scan. current height:%v", maxBlockHeight)
			break
		}

		//扫描下一个区块
		curBlockHeight += 1
		bs.wm.Log.Infof("block scanner try to scan block No.%v", curBlockHeight)

		curBlock, err := bs.wm.GetBlockByNum(curBlockHeight, true)
		if err != nil {
			bs.wm.Log.Errorf("EthGetBlockSpecByBlockNum failed, err = %v", err)
			break
		}

		isFork := false

		if curBlock.PreviousHash != curBlockHash {
			previousHeight = curBlockHeight - 1
			bs.wm.Log.Infof("block has been fork on height: %v.", curBlockHeight)
			bs.wm.Log.Infof("block height: %v local hash = %v ", previousHeight, curBlockHash)
			bs.wm.Log.Infof("block height: %v mainnet hash = %v ", previousHeight, curBlock.PreviousHash)

			bs.wm.Log.Infof("delete recharge records on block height: %v.", previousHeight)

			//查询本地分叉的区块
			forkBlock, _ := bs.GetLocalBlock(previousHeight)

			bs.DeleteUnscanRecord(previousHeight)

			curBlockHeight = previousHeight - 1 //倒退2个区块重新扫描

			curBlock, err = bs.GetLocalBlock(curBlockHeight)
			if err != nil {
				bs.wm.Log.Std.Error("block scanner can not get local block; unexpected error: %v", err)
				bs.wm.Log.Info("block scanner prev block height:", curBlockHeight)

				curBlock, err = bs.wm.GetBlockByNum(curBlockHeight, false)
				if err != nil {
					bs.wm.Log.Errorf("EthGetBlockSpecByBlockNum  failed, block number=%v, err=%v", curBlockHeight, err)
					break
				}

			}

			curBlockHash = curBlock.BlockHash
			bs.wm.Log.Infof("rescan block on height:%v, hash:%v.", curBlockHeight, curBlockHash)

			err = bs.SaveLocalBlockHead(curBlock.BlockHeight, curBlock.BlockHash)
			if err != nil {
				bs.wm.Log.Errorf("save local block unscaned failed, err=%v", err)
				break
			}

			isFork = true

			if forkBlock != nil {

				//通知分叉区块给观测者，异步处理
				bs.newBlockNotify(forkBlock, isFork)
			}

		} else {
			err = bs.BatchExtractTransaction(curBlock.BlockHeight, curBlock.Transactions)
			if err != nil {
				bs.wm.Log.Errorf("block scanner can not extractRechargeRecords; unexpected error: %v", err)
				break
			}

			bs.SaveLocalBlockHead(curBlock.BlockHeight, curBlock.BlockHash)
			bs.SaveLocalBlock(curBlock)

			isFork = false

			bs.newBlockNotify(curBlock, isFork)
		}

		curBlockHeight = curBlock.BlockHeight
		curBlockHash = curBlock.BlockHash

	}

	bs.RescanFailedRecord()
}

//newExtractDataNotify 发送通知
func (bs *BlockScanner) newExtractDataNotify(height uint64, extractDataList map[string][]*openwallet.TxExtractData, extractContractData map[string]*openwallet.SmartContractReceipt) error {

	for o, _ := range bs.Observers {
		for key, extractData := range extractDataList {
			for _, data := range extractData {
				err := o.BlockExtractDataNotify(key, data)
				if err != nil {
					//记录未扫区块
					reason := fmt.Sprintf("ExtractData Notify failed: %s", bs.wm.Symbol())
					err = bs.SaveUnscannedTransaction(height, reason)
					if err != nil {
						bs.wm.Log.Errorf("block height: %d, save unscan record failed. unexpected error: %v", height, err.Error())
						return err
					}
				}
			}
		}

		for key, data := range extractContractData {
			err := o.BlockExtractSmartContractDataNotify(key, data)
			if err != nil {
				//记录未扫区块
				reason := fmt.Sprintf("ExtractContractData Notify failed: %s", bs.wm.Symbol())
				err = bs.SaveUnscannedTransaction(height, reason)
				if err != nil {
					bs.wm.Log.Errorf("block height: %d, save unscan record failed. unexpected error: %v", height, err.Error())
					return err
				}
			}
		}
	}

	return nil
}

//BatchExtractTransaction 批量提取交易单
func (bs *BlockScanner) BatchExtractTransaction(height uint64, txs []*BlockTransaction) error {

	var (
		quit       = make(chan struct{})
		done       = 0 //完成标记
		failed     = 0
		shouldDone = len(txs) //需要完成的总数
	)

	if len(txs) == 0 {
		return fmt.Errorf("BatchExtractTransaction block is nil.")
	}

	//生产通道
	producer := make(chan ExtractResult)
	defer close(producer)

	//消费通道
	worker := make(chan ExtractResult)
	defer close(worker)

	//保存工作
	saveWork := func(height uint64, result chan ExtractResult) {
		//回收创建的地址
		for gets := range result {

			if gets.Success {

				notifyErr := bs.newExtractDataNotify(height, gets.extractData, gets.extractContractData)
				//saveErr := bs.SaveRechargeToWalletDB(height, gets.Recharges)
				if notifyErr != nil {
					failed++ //标记保存失败数
					bs.wm.Log.Std.Info("newExtractDataNotify unexpected error: %v", notifyErr)
				}

			} else {
				//记录未扫区块
				unscanRecord := openwallet.NewUnscanRecord(height, "", "", bs.wm.Symbol())
				bs.SaveUnscanRecord(unscanRecord)
				bs.wm.Log.Std.Info("block height: %d extract failed.", height)
				failed++ //标记保存失败数
			}
			//累计完成的线程数
			done++
			if done == shouldDone {
				//bs.wm.Log.Std.Info("done = %d, shouldDone = %d ", done, len(txs))
				close(quit) //关闭通道，等于给通道传入nil
			}
		}
	}

	//提取工作
	extractWork := func(mTxs []*BlockTransaction, eProducer chan ExtractResult) {
		for _, tx := range mTxs {
			bs.extractingCH <- struct{}{}
			//shouldDone++
			go func(mTx *BlockTransaction, end chan struct{}, mProducer chan<- ExtractResult) {
				mTx.FilterFunc = bs.ScanTargetFuncV2
				mTx.BlockHeight = height
				//导出提出的交易
				mProducer <- bs.ExtractTransaction(mTx)
				//释放
				<-end

			}(tx, bs.extractingCH, eProducer)
		}
	}

	/*	开启导出的线程	*/

	//独立线程运行消费
	go saveWork(height, worker)

	//独立线程运行生产
	go extractWork(txs, producer)

	//以下使用生产消费模式
	bs.extractRuntime(producer, worker, quit)

	if failed > 0 {
		return fmt.Errorf("block scanner saveWork failed")
	} else {
		return nil
	}

	//return nil
}

//extractRuntime 提取运行时
func (bs *BlockScanner) extractRuntime(producer chan ExtractResult, worker chan ExtractResult, quit chan struct{}) {

	var (
		values = make([]ExtractResult, 0)
	)

	for {

		var activeWorker chan<- ExtractResult
		var activeValue ExtractResult

		//当数据队列有数据时，释放顶部，传输给消费者
		if len(values) > 0 {
			activeWorker = worker
			activeValue = values[0]

		}

		select {

		//生成者不断生成数据，插入到数据队列尾部
		case pa := <-producer:
			values = append(values, pa)
		case <-quit:
			//退出
			//bs.wm.Log.Std.Info("block scanner have been scanned!")
			return
		case activeWorker <- activeValue:
			//wm.Log.Std.Info("Get %d", len(activeValue))
			values = values[1:]
		}
	}

}

// UpdateTxByReceipt
func (bs *BlockScanner) UpdateTxByReceipt(tx *BlockTransaction) error {
	//过滤掉未打包交易
	if tx.BlockHeight == 0 || tx.BlockHash == "" {
		return nil
	}

	//获取交易回执
	txReceipt, err := bs.wm.GetTransactionReceipt(tx.Hash)
	if err != nil {
		bs.wm.Log.Errorf("get transaction receipt failed, err: %v", err)
		return err
	}
	tx.receipt = txReceipt
	tx.Gas = common.NewString(txReceipt.ETHReceipt.GasUsed).String()
	tx.Status = txReceipt.ETHReceipt.Status
	tx.decimal = bs.wm.Decimal()

	return nil
}

// GetBalanceByAddress 获取地址余额
func (bs *BlockScanner) GetBalanceByAddress(address ...string) ([]*openwallet.Balance, error) {
	type addressBalance struct {
		Address string
		Index   uint64
		Balance *openwallet.Balance
	}

	threadControl := make(chan int, 20)
	defer close(threadControl)
	resultChan := make(chan *addressBalance, 1024)
	defer close(resultChan)
	done := make(chan int, 1)
	count := len(address)
	resultBalance := make([]*openwallet.Balance, count)
	resultSaveFailed := false
	//save result
	go func() {
		for i := 0; i < count; i++ {
			addr := <-resultChan
			if addr.Balance != nil {
				resultBalance[addr.Index] = addr.Balance
			} else {
				resultSaveFailed = true
			}
		}
		done <- 1
	}()

	query := func(addr *addressBalance) {
		threadControl <- 1
		defer func() {
			resultChan <- addr
			<-threadControl
		}()

		balanceConfirmed, err := bs.wm.GetAddrBalance(AppendOxToAddress(addr.Address), "latest")
		if err != nil {
			bs.wm.Log.Error("get address[", addr.Address, "] balance failed, err=", err)
			return
		}

		balanceAll, err := bs.wm.GetAddrBalance(AppendOxToAddress(addr.Address), "pending")
		if err != nil {
			balanceAll = balanceConfirmed
		}

		balanceUnconfirmed := big.NewInt(0)
		balanceUnconfirmed.Sub(balanceAll, balanceConfirmed)

		balance := &openwallet.Balance{
			Symbol:  bs.wm.Symbol(),
			Address: addr.Address,
		}
		confirmed := common.BigIntToDecimals(balanceConfirmed, bs.wm.Decimal())
		all := common.BigIntToDecimals(balanceAll, bs.wm.Decimal())
		unconfirmed := common.BigIntToDecimals(balanceUnconfirmed, bs.wm.Decimal())

		balance.Balance = all.String()
		balance.UnconfirmBalance = unconfirmed.String()
		balance.ConfirmBalance = confirmed.String()
		addr.Balance = balance
	}

	for i, _ := range address {
		addrbl := &addressBalance{
			Address: address[i],
			Index:   uint64(i),
		}
		go query(addrbl)
	}

	<-done
	if resultSaveFailed {
		return nil, fmt.Errorf("get balance of addresses failed. ")
	}
	return resultBalance, nil
}

// ExtractTransaction 提取交易单
func (bs *BlockScanner) ExtractTransaction(tx *BlockTransaction) ExtractResult {
	var (
		result = ExtractResult{
			BlockHeight:         tx.BlockHeight,
			TxID:                tx.Hash,
			extractData:         make(map[string][]*openwallet.TxExtractData),
			extractContractData: make(map[string]*openwallet.SmartContractReceipt),
			Success:             true,
		}
	)

	if tx.BlockNumber == "" {
		result.Success = false
		return result
	}

	//获取交易回执
	err := bs.UpdateTxByReceipt(tx)
	if err != nil {
		result.Success = false
		return result
	}

	// 提取转账交易单
	bs.extractBaseTransaction(tx, &result)

	// 提取智能合约交易单
	bs.extractSmartContractTransaction(tx, &result)

	return result
}

// extractBaseTransaction 提取转账交易单
func (bs *BlockScanner) extractBaseTransaction(tx *BlockTransaction, result *ExtractResult) {

	tokenEvent := tx.receipt.ParseTransferEvent()

	isTokenTransfer := false
	if len(tokenEvent) > 0 {
		isTokenTransfer = true
	}

	//提出主币交易单
	extractData := bs.extractETHTransaction(tx, isTokenTransfer)
	for sourceKey, data := range extractData {
		extractDataArray := result.extractData[sourceKey]
		if extractDataArray == nil {
			extractDataArray = make([]*openwallet.TxExtractData, 0)
		}
		extractDataArray = append(extractDataArray, data)
		result.extractData[sourceKey] = extractDataArray
	}

	//提取代币交易单
	for contractAddress, tokenEventArray := range tokenEvent {
		//提出主币交易单
		extractERC20Data := bs.extractERC20Transaction(tx, contractAddress, tokenEventArray)
		for sourceKey, data := range extractERC20Data {
			extractDataArray := result.extractData[sourceKey]
			if extractDataArray == nil {
				extractDataArray = make([]*openwallet.TxExtractData, 0)
			}
			extractDataArray = append(extractDataArray, data)
			result.extractData[sourceKey] = extractDataArray
		}
	}
}

//extractETHTransaction 提取主币交易单
func (bs *BlockScanner) extractETHTransaction(tx *BlockTransaction, isTokenTransfer bool) map[string]*openwallet.TxExtractData {

	txExtractMap := make(map[string]*openwallet.TxExtractData)
	from := tx.From
	to := tx.To
	status := common.NewString(tx.Status).String()
	reason := ""
	nowUnix := time.Now().Unix()
	txType := uint64(0)

	coin := openwallet.Coin{
		Symbol:     bs.wm.Symbol(),
		IsContract: false,
	}

	if isTokenTransfer {
		txType = 1
	}

	ethAmount := tx.GetAmountEthString()
	feeprice := tx.GetTxFeeEthString()

	targetResult := tx.FilterFunc(openwallet.ScanTargetParam{
		ScanTarget:     from,
		Symbol:         bs.wm.Symbol(),
		ScanTargetType: openwallet.ScanTargetTypeAccountAddress})
	if targetResult.Exist {
		input := &openwallet.TxInput{}
		input.TxID = tx.Hash
		input.Address = from
		input.Amount = ethAmount
		input.Coin = coin
		input.Index = 0
		input.Sid = openwallet.GenTxInputSID(tx.Hash, bs.wm.Symbol(), "", 0)
		input.CreateAt = nowUnix
		input.BlockHeight = tx.BlockHeight
		input.BlockHash = tx.BlockHash
		input.TxType = txType

		//transactions = append(transactions, &transaction)

		ed := txExtractMap[targetResult.SourceKey]
		if ed == nil {
			ed = openwallet.NewBlockExtractData()
			txExtractMap[targetResult.SourceKey] = ed
		}

		ed.TxInputs = append(ed.TxInputs, input)

		//手续费作为一个输入
		feeInput := &openwallet.TxInput{}
		feeInput.Recharge.Sid = openwallet.GenTxInputSID(tx.Hash, bs.wm.Symbol(), "", uint64(1))
		feeInput.Recharge.TxID = tx.Hash
		feeInput.Recharge.Address = from
		feeInput.Recharge.Coin = coin
		feeInput.Recharge.Amount = feeprice
		feeInput.Recharge.BlockHash = tx.BlockHash
		feeInput.Recharge.BlockHeight = tx.BlockHeight
		feeInput.Recharge.Index = 1 //账户模型填0
		feeInput.Recharge.CreateAt = nowUnix
		feeInput.Recharge.TxType = txType

		ed.TxInputs = append(ed.TxInputs, feeInput)

	}

	targetResult2 := tx.FilterFunc(openwallet.ScanTargetParam{
		ScanTarget:     to,
		Symbol:         bs.wm.Symbol(),
		ScanTargetType: openwallet.ScanTargetTypeAccountAddress})
	if targetResult2.Exist {
		output := &openwallet.TxOutPut{}
		output.TxID = tx.Hash
		output.Address = to
		output.Amount = ethAmount
		output.Coin = coin
		output.Index = 0
		output.Sid = openwallet.GenTxInputSID(tx.Hash, bs.wm.Symbol(), "", 0)
		output.CreateAt = nowUnix
		output.BlockHeight = tx.BlockHeight
		output.BlockHash = tx.BlockHash
		output.TxType = txType

		ed := txExtractMap[targetResult2.SourceKey]
		if ed == nil {
			ed = openwallet.NewBlockExtractData()
			txExtractMap[targetResult2.SourceKey] = ed
		}

		ed.TxOutputs = append(ed.TxOutputs, output)
	}

	for _, extractData := range txExtractMap {

		tx := &openwallet.Transaction{
			Fees:        feeprice,
			Coin:        coin,
			BlockHash:   tx.BlockHash,
			BlockHeight: tx.BlockHeight,
			TxID:        tx.Hash,
			Decimal:     bs.wm.Decimal(),
			Amount:      ethAmount,
			ConfirmTime: nowUnix,
			From:        []string{from + ":" + ethAmount},
			To:          []string{to + ":" + ethAmount},
			Status:      status,
			Reason:      reason,
			TxType:      txType,
		}

		wxID := openwallet.GenTransactionWxID(tx)
		tx.WxID = wxID
		extractData.Transaction = tx

	}
	return txExtractMap
}

//extractERC20Transaction
func (bs *BlockScanner) extractERC20Transaction(tx *BlockTransaction, contractAddress string, tokenEvent []*TransferEvent) map[string]*openwallet.TxExtractData {

	nowUnix := time.Now().Unix()
	status := common.NewString(tx.Status).String()
	reason := ""
	txExtractMap := make(map[string]*openwallet.TxExtractData)

	contractId := openwallet.GenContractID(bs.wm.Symbol(), contractAddress)
	coin := openwallet.Coin{
		Symbol:     bs.wm.Symbol(),
		IsContract: true,
		ContractID: contractId,
		Contract: openwallet.SmartContract{
			ContractID: contractId,
			Address:    contractAddress,
			Symbol:     bs.wm.Symbol(),
		},
	}

	//提取出账部分记录
	from := bs.extractERC20Detail(tx, contractAddress, tokenEvent, true, txExtractMap)

	//提取入账部分记录
	to := bs.extractERC20Detail(tx, contractAddress, tokenEvent, false, txExtractMap)

	for _, extractData := range txExtractMap {
		tx := &openwallet.Transaction{
			Fees:        "0",
			Coin:        coin,
			BlockHash:   tx.BlockHash,
			BlockHeight: tx.BlockHeight,
			TxID:        tx.Hash,
			Amount:      "0",
			ConfirmTime: nowUnix,
			From:        from,
			To:          to,
			Status:      status,
			Reason:      reason,
			TxType:      0,
		}

		wxID := openwallet.GenTransactionWxID(tx)
		tx.WxID = wxID
		extractData.Transaction = tx

	}
	return txExtractMap
}

//extractERC20Detail
func (bs *BlockScanner) extractERC20Detail(tx *BlockTransaction, contractAddress string, tokenEvent []*TransferEvent, isInput bool, extractData map[string]*openwallet.TxExtractData) []string {

	var (
		addrs  = make([]string, 0)
		txType = uint64(0)
	)

	contractId := openwallet.GenContractID(bs.wm.Symbol(), contractAddress)
	coin := openwallet.Coin{
		Symbol:     bs.wm.Symbol(),
		IsContract: true,
		ContractID: contractId,
		Contract: openwallet.SmartContract{
			ContractID: contractId,
			Address:    contractAddress,
			Symbol:     bs.wm.Symbol(),
		},
	}

	createAt := time.Now().Unix()
	for i, te := range tokenEvent {

		address := ""
		if isInput {
			address = te.TokenFrom
		} else {
			address = te.TokenTo
		}

		targetResult := tx.FilterFunc(openwallet.ScanTargetParam{
			ScanTarget:     address,
			Symbol:         bs.wm.Symbol(),
			ScanTargetType: openwallet.ScanTargetTypeAccountAddress})
		if targetResult.Exist {

			detail := openwallet.Recharge{}
			detail.Sid = openwallet.GenTxInputSID(tx.Hash, bs.wm.Symbol(), coin.ContractID, uint64(i))
			detail.TxID = tx.Hash
			detail.Address = address
			detail.Coin = coin
			detail.Amount = te.Value.String()
			detail.BlockHash = tx.BlockHash
			detail.BlockHeight = tx.BlockHeight
			detail.Index = uint64(i) //账户模型填0
			detail.CreateAt = createAt
			detail.TxType = txType

			ed := extractData[targetResult.SourceKey]
			if ed == nil {
				ed = openwallet.NewBlockExtractData()
				extractData[targetResult.SourceKey] = ed
			}

			if isInput {
				txInput := &openwallet.TxInput{Recharge: detail}
				ed.TxInputs = append(ed.TxInputs, txInput)
			} else {
				txOutPut := &openwallet.TxOutPut{Recharge: detail}
				ed.TxOutputs = append(ed.TxOutputs, txOutPut)
			}

		}

		addrs = append(addrs, address+":"+te.Value.String())

	}
	return addrs
}

// extractSmartContractTransaction 提取智能合约交易单
func (bs *BlockScanner) extractSmartContractTransaction(tx *BlockTransaction, result *ExtractResult) {

	contractAddress := strings.ToLower(tx.To)

	//查找合约是否存在
	targetResult := tx.FilterFunc(openwallet.ScanTargetParam{
		ScanTarget:     contractAddress,
		Symbol:         bs.wm.Symbol(),
		ScanTargetType: openwallet.ScanTargetTypeContractAddress})
	if !targetResult.Exist {
		return //不存在返回
	}

	//查找合约对象信息
	contract, ok := targetResult.TargetInfo.(*openwallet.SmartContract)
	if !ok {
		result.Success = false
		return
	}

	coin := openwallet.Coin{
		Symbol:     bs.wm.Symbol(),
		IsContract: true,
		ContractID: contract.ContractID,
		Contract:   *contract,
	}

	createAt := time.Now().Unix()

	//迭代每个日志，提取时间日志
	events := make([]*openwallet.SmartContractEvent, 0)
	for _, log := range tx.receipt.ETHReceipt.Logs {

		logContractAddress := strings.ToLower(log.Address.String())

		logTargetResult := tx.FilterFunc(openwallet.ScanTargetParam{
			ScanTarget:     logContractAddress,
			Symbol:         bs.wm.Symbol(),
			ScanTargetType: openwallet.ScanTargetTypeContractAddress})
		if !logTargetResult.Exist {
			continue
		}
		logContract, logOk := logTargetResult.TargetInfo.(*openwallet.SmartContract)
		if !logOk {
			bs.wm.Log.Errorf("log target result can not convert to openwallet.SmartContract")
			result.Success = false
			return
		}

		abiInstance, logErr := abi.JSON(strings.NewReader(logContract.GetABI()))
		if logErr != nil {
			bs.wm.Log.Errorf("abi decode json failed, err: %v", logErr)
			result.Success = false
			return
		}

		_, eventName, logJSON, logErr := bs.wm.DecodeReceiptLogResult(abiInstance, *log)
		if logErr != nil {
			bs.wm.Log.Errorf("DecodeReceiptLogResult failed, err: %v", logErr)
			//result.Success = false
			//return
			continue
		}

		e := &openwallet.SmartContractEvent{
			Contract: logContract,
			Event:    eventName,
			Value:    logJSON,
		}

		events = append(events, e)

	}

	scReceipt := &openwallet.SmartContractReceipt{
		Coin:        coin,
		TxID:        tx.Hash,
		From:        tx.From,
		RawReceipt:  tx.receipt.Raw,
		Events:      events,
		BlockHash:   tx.BlockHash,
		BlockHeight: tx.BlockHeight,
		ConfirmTime: createAt,
		Status:      common.NewString(tx.Status).String(),
		Reason:      "",
	}

	scReceipt.GenWxID()

	result.extractContractData[targetResult.SourceKey] = scReceipt

}

//ExtractTransactionData 扫描一笔交易
func (bs *BlockScanner) ExtractTransactionData(txid string, scanTargetFunc openwallet.BlockScanTargetFunc) (map[string][]*openwallet.TxExtractData, error) {
	//result := bs.ExtractTransaction(0, "", txid, scanAddressFunc)
	tx, err := bs.wm.GetTransactionByHash(txid)
	if err != nil {
		bs.wm.Log.Errorf("get transaction by has failed, err=%v", err)
		return nil, fmt.Errorf("get transaction by has failed, err=%v", err)
	}
	tx.FilterFunc = func(target openwallet.ScanTargetParam) openwallet.ScanTargetResult {
		sourceKey, ok := scanTargetFunc(openwallet.ScanTarget{
			Address:          target.ScanTarget,
			Symbol:           bs.wm.Symbol(),
			BalanceModelType: bs.wm.BalanceModelType(),
		})
		return openwallet.ScanTargetResult{
			SourceKey: sourceKey,
			Exist:     ok,
		}
	}
	result := bs.ExtractTransaction(tx)
	return result.extractData, nil
}

//ExtractTransactionAndReceiptData 提取交易单及交易回执数据
//@required
func (bs *BlockScanner) ExtractTransactionAndReceiptData(txid string, scanTargetFunc openwallet.BlockScanTargetFuncV2) (map[string][]*openwallet.TxExtractData, map[string]*openwallet.SmartContractReceipt, error) {
	//result := bs.ExtractTransaction(0, "", txid, scanAddressFunc)
	tx, err := bs.wm.GetTransactionByHash(txid)
	if err != nil {
		bs.wm.Log.Errorf("get transaction by has failed, err=%v", err)
		return nil, nil, fmt.Errorf("get transaction by has failed, err=%v", err)
	}
	tx.FilterFunc = scanTargetFunc
	result := bs.ExtractTransaction(tx)
	return result.extractData, result.extractContractData, nil
}

//GetScannedBlockHeader 获取当前已扫区块高度
func (bs *BlockScanner) GetScannedBlockHeader() (*openwallet.BlockHeader, error) {

	var (
		blockHeight uint64 = 0
		hash        string
		err         error
	)

	blockHeight, hash, err = bs.GetLocalBlockHead()
	if err != nil {
		bs.wm.Log.Errorf("get local new block failed, err=%v", err)
		return nil, err
	}

	//如果本地没有记录，查询接口的高度
	if blockHeight == 0 {
		blockHeight, err = bs.wm.GetBlockNumber()
		if err != nil {
			bs.wm.Log.Errorf("EthGetBlockNumber failed, err=%v", err)
			return nil, err
		}

		//就上一个区块链为当前区块
		blockHeight = blockHeight - 1

		block, err := bs.wm.GetBlockByNum(blockHeight, false)
		if err != nil {
			bs.wm.Log.Errorf("get block spec by block number failed, err=%v", err)
			return nil, err
		}
		hash = block.BlockHash
	}

	return &openwallet.BlockHeader{Height: blockHeight, Hash: hash}, nil
}

//GetCurrentBlockHeader 获取当前区块高度
func (bs *BlockScanner) GetCurrentBlockHeader() (*openwallet.BlockHeader, error) {

	var (
		blockHeight uint64 = 0
		hash        string
		err         error
	)

	blockHeight, err = bs.wm.GetBlockNumber()
	if err != nil {
		bs.wm.Log.Errorf("EthGetBlockNumber failed, err=%v", err)
		return nil, err
	}

	block, err := bs.wm.GetBlockByNum(blockHeight, false)
	if err != nil {
		bs.wm.Log.Errorf("get block spec by block number failed, err=%v", err)
		return nil, err
	}
	hash = block.BlockHash

	return &openwallet.BlockHeader{Height: blockHeight, Hash: hash}, nil
}

func (bs *BlockScanner) GetGlobalMaxBlockHeight() uint64 {

	maxBlockHeight, err := bs.wm.GetBlockNumber()
	if err != nil {
		bs.wm.Log.Errorf("get max height of eth failed, err=%v", err)
		return 0
	}
	return maxBlockHeight
}

func (bs *BlockScanner) SaveUnscannedTransaction(blockHeight uint64, reason string) error {
	unscannedRecord := &openwallet.UnscanRecord{
		BlockHeight: blockHeight,
		Reason:      reason,
		Symbol:      bs.wm.Symbol(),
	}
	return bs.SaveUnscanRecord(unscannedRecord)
}
