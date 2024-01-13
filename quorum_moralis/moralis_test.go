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
	"github.com/blocktree/openwallet/v2/log"
	"testing"
)

var (
	ts *MoralisSDK
)

func init() {

	ts = New(
		"",
		"polygon",
		true)
}

func TestMoralisSDK_GetBlockByMoralis(t *testing.T) {
	result, err := ts.GetBlockByMoralis(4196335)
	if err != nil {
		t.Errorf("GetBlockByMoralis failed, err: %v", err)
		return
	}
	log.Debugf("result: %v", result)
}
