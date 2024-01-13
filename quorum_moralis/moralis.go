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

package quorum_moralis

import (
	"fmt"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/imroc/req"
	"github.com/tidwall/gjson"
)

const MoralisApiUrl = "https://deep-index.moralis.io/api/v2.2/"

type MoralisSDK struct {
	APIKey string
	Chain  string
	Debug  bool
}

func New(apiKey, chain string, debug bool) *MoralisSDK {
	client := &MoralisSDK{APIKey: apiKey, Chain: chain, Debug: debug}
	return client
}

func (sdk *MoralisSDK) get(url string) (*gjson.Result, error) {
	authHeader := req.Header{
		"Accept":    "application/json",
		"X-API-Key": sdk.APIKey,
	}

	r, err := req.Get(url, authHeader)

	if sdk.Debug {
		log.Debugf("%+v\n", r)
	}

	if err != nil {
		return nil, err
	}

	resp := gjson.ParseBytes(r.Bytes())
	return &resp, nil
}

func (sdk *MoralisSDK) GetBlockByMoralis(blockNum uint64) (*gjson.Result, error) {
	url := fmt.Sprintf("%s/block/%d?chain=%s&include=internal_transactions", MoralisApiUrl, blockNum, sdk.Chain)
	result, err := sdk.get(url)
	if err != nil {
		return nil, err
	}
	return result, nil
}
