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
	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/v2/common/file"
	"math/big"
	"path/filepath"
	"strings"
)

const (
	Symbol    = "QUORUM"
	CurveType = owcrypt.ECC_CURVE_SECP256K1
)

type WalletConfig struct {

	//币种
	Symbol string
	//本地数据库文件路径
	DBPath string
	//钱包服务API
	ServerAPI string
	//曲线类型
	CurveType uint32
	//网络ID
	ChainID uint64
	//数据目录
	DataDir string
	//固定gasLimit值
	FixGasLimit *big.Int
	//固定gasPrice值
	FixGasPrice *big.Int
	//补偿gasPrice值
	OffsetsGasPrice *big.Int
	//nonce计算方式, 0: auto-increment nonce, 1: latest nonce
	NonceComputeMode int64
	//Broadcast node RPC API
	BroadcastAPI string
	// Use QuickNode Single Flight RPC
	UseQNSingleFlightRPC int64
	// Detect unknown contracts
	DetectUnknownContracts int64
}

func NewConfig(symbol string) *WalletConfig {
	c := WalletConfig{}
	c.Symbol = symbol
	c.CurveType = CurveType
	return &c
}

// 创建文件夹
func (wc *WalletConfig) makeDataDir() {

	if len(wc.DataDir) == 0 {
		//默认路径当前文件夹./data
		wc.DataDir = "data"
	}

	//本地数据库文件路径
	wc.DBPath = filepath.Join(wc.DataDir, strings.ToLower(wc.Symbol), "db")

	//创建目录
	file.MkdirAll(wc.DBPath)
}
