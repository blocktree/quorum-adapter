/*
 * Copyright 2019 The openwallet Authors
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

package quorumtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"strings"
	"testing"
)

var (
	abiJSON = `
[{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"amount","type":"uint256"}],"name":"withdrawEther","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_value","type":"uint256"}],"name":"burn","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_value","type":"uint256"}],"name":"unfreeze","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"freezeOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_value","type":"uint256"}],"name":"freeze","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"},{"name":"","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"inputs":[{"name":"initialSupply","type":"uint256"},{"name":"tokenName","type":"string"},{"name":"decimalUnits","type":"uint8"},{"name":"tokenSymbol","type":"string"}],"payable":false,"type":"constructor"},{"payable":true,"type":"fallback"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Burn","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Freeze","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Unfreeze","type":"event"}]

`
	dataJSON = `
	{
        "blockHash": "0x327723fbf6e7452be9c134a955c7d9ec549479c3c22d783b6c264153a5e9da2d",
        "blockNumber": "0x910baa",
        "contractAddress": null,
        "cumulativeGasUsed": "0x6ef978",
        "from": "0xffe3db266852a7e09123d913371fceb62e6ecd70",
        "gasUsed": "0xd2c2",
        "logs": [
            {
                "address": "0xb8c77482e45f1f44de1745f52c74426c631bdd52",
                "topics": [
                    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
                    "0x000000000000000000000000ffe3db266852a7e09123d913371fceb62e6ecd70",
                    "0x00000000000000000000000049a453ce4dca928c4e92017d17292a209c485a00"
                ],
                "data": "0x000000000000000000000000000000000000000000000000f84b4b41a6a1b400",
                "blockNumber": "0x910baa",
                "transactionHash": "0x17a63396753b17b6fdc22a8117f9dfbf818a257dfd08497d8675148d0a1f701b",
                "transactionIndex": "0x9e",
                "blockHash": "0x327723fbf6e7452be9c134a955c7d9ec549479c3c22d783b6c264153a5e9da2d",
                "logIndex": "0x8a",
                "removed": false
            }
        ],
        "logsBloom": "0x00400008000000000000404000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000002000000000000000004000000000000000000000000000000000000000000000000000",
        "status": "0x1",
        "to": "0xb8c77482e45f1f44de1745f52c74426c631bdd52",
        "transactionHash": "0x17a63396753b17b6fdc22a8117f9dfbf818a257dfd08497d8675148d0a1f701b",
        "transactionIndex": "0x9e"
    }
`
)

func TestReceiptUnpackLog(t *testing.T) {

	type EventTransfer struct {
		From  common.Address
		To    common.Address
		Value *big.Int
	}

	var transfer EventTransfer

	var receipt types.Receipt
	receipt.UnmarshalJSON([]byte(dataJSON))
	abi, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Fatal(err)
		return
	}
	bc := bind.NewBoundContract(common.HexToAddress("0x0"), abi, nil, nil, nil)
	err = bc.UnpackLog(&transfer, "Transfer", *receipt.Logs[0])
	if err != nil {
		t.Fatal(err)
		return
	}
	fmt.Printf("from: %s \n", transfer.From.String())
	fmt.Printf("to: %s \n", transfer.To.String())
	fmt.Printf("value: %+v \n", transfer.Value.String())
}

func TestReceiptUnpackLogIntoMap(t *testing.T) {

	var receipt types.Receipt
	receipt.UnmarshalJSON([]byte(dataJSON))
	abi, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Fatal(err)
		return
	}

	bc := bind.NewBoundContract(common.HexToAddress("0x0"), abi, nil, nil, nil)
	event, err := abi.EventByID(receipt.Logs[0].Topics[0])
	if err != nil {
		t.Fatal(err)
		return
	}
	transfer := make(map[string]interface{})
	//fmt.Printf("event name: %s \n", name)
	err = bc.UnpackLogIntoMap(transfer, event.Name, *receipt.Logs[0])
	if err != nil {
		t.Fatal(err)
		return
	}
	for _, input := range event.Inputs {
		//fmt.Printf("input.Name: %s \n", input.Name)
		//fmt.Printf("input.Type: %s \n", input.Type.String())
		typeName := input.Type.String()
		var value interface{}
		switch typeName {
		case "address":
			value = transfer[input.Name].(common.Address).String()
		case "uint256":
			value = transfer[input.Name].(*big.Int).String()
		}
		fmt.Printf("%s: %v \n", input.Name, value)
	}
}

func TestCallResultUnpackIntoMap(t *testing.T) {

	const simpleTuple = `[{"name":"tuple","constant":false,"outputs":[{"type":"tuple","name":"ret","components":[{"type":"int256","name":"a"},{"type":"int256","name":"b"},{"type":"address","name":"c"}]}]}]`
	abi, err := abi.JSON(strings.NewReader(simpleTuple))
	if err != nil {
		t.Fatal(err)
		return
	}
	buff := new(bytes.Buffer)

	buff.Write(common.Hex2Bytes("00000000000000000000000000000000000000000000FFFFFFFFFFFFFFFFFF01")) // ret[a] = 1
	buff.Write(common.Hex2Bytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")) // ret[b] = -1
	buff.Write(common.Hex2Bytes("000000000000000000000000B8c77482e45F1F44dE1745F52C74426C631bDD52")) // ret[c] = 0xffe3db266852a7e09123d913371fceb62e6ecd70

	result := make(map[string]interface{})

	err = abi.UnpackIntoMap(result, "tuple", buff.Bytes())
	if err != nil {
		t.Fatal(err)
		return
	}
	t.Logf("result: %+v \n", result)
	jsonResult, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
		return
	}
	t.Logf("jsonResult: %s \n", jsonResult)
}
