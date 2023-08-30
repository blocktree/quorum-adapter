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
package quorum_rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/imroc/req"
	"github.com/tidwall/gjson"
)

type Client struct {
	context context.Context

	BaseURL      string
	BroadcastURL string
	Debug        bool
	RawClient    *rpc.Client //原生ETH客户端
}

func Dial(baseURL, broadcastURL string, debug bool) (*Client, error) {
	context := context.Background()
	client := &Client{BaseURL: baseURL, BroadcastURL: broadcastURL, Debug: debug}
	rawClient, err := rpc.DialContext(context, baseURL)
	if err != nil {
		return nil, err
	}
	client.RawClient = rawClient
	client.context = context
	return client, nil
}

func (c *Client) Call(method string, params []interface{}) (*gjson.Result, error) {

	if method == "eth_sendRawTransaction" && len(c.BroadcastURL) != 0 {
		// 广播交易使用BroadcastURL的节点
		return c.callByHttpClient(c.BroadcastURL, method, params)
	} else {
		//return c.callByETHClient(method, params)
		return c.callByHttpClient(c.BaseURL, method, params)
	}
}

func (c *Client) callByHttpClient(url, method string, params []interface{}) (*gjson.Result, error) {
	authHeader := req.Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}
	body := make(map[string]interface{}, 0)
	body["jsonrpc"] = "2.0"
	body["id"] = 1
	body["method"] = method
	body["params"] = params

	r, err := req.Post(url, req.BodyJSON(&body), authHeader)

	if c.Debug {
		log.Debugf("%+v\n", r)
	}

	if err != nil {
		return nil, err
	}

	resp := gjson.ParseBytes(r.Bytes())
	err = isError(&resp)
	if err != nil {
		return nil, err
	}

	result := resp.Get("result")

	return &result, nil
}

func (c *Client) callByETHClient(method string, params []interface{}) (*gjson.Result, error) {

	var res interface{}
	err := c.RawClient.CallContext(c.context, &res, method, params...)
	if err != nil {
		return nil, err
	}

	r, _ := json.Marshal(map[string]interface{}{
		"result": res,
	})

	resp := gjson.ParseBytes(r)
	result := resp.Get("result")

	return &result, nil
}

// isError 是否报错
func isError(result *gjson.Result) error {
	var (
		err error
	)

	if !result.Get("error").IsObject() {

		if !result.Get("result").Exists() {
			return fmt.Errorf("Response is empty! ")
		}

		return nil
	}

	errInfo := fmt.Sprintf("[%d]%s",
		result.Get("error.code").Int(),
		result.Get("error.message").String())
	err = errors.New(errInfo)

	return err
}
