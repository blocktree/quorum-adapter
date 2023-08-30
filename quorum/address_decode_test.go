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
	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/v2/log"
	"testing"
)

func TestAddressDecoder_PublicKeyToAddress(t *testing.T) {
	pub, _ := hex.DecodeString("0265ff85a638b555ad5f15359ef0d80688452bd4dae3a29ecdf53e74b76862a6f2")

	decoder := AddressDecoder{}

	addr, err := decoder.PublicKeyToAddress(pub, false)
	if err != nil {
		t.Errorf("AddressDecode failed unexpected error: %v\n", err)
		return
	}
	t.Logf("addr: %s", addr)
	//	0xa8a4b2d37c591db3310df648942bf3351cecd984
}

func TestAddressDecoder_PrivateKeyToAddress(t *testing.T) {
	prv, _ := hex.DecodeString("0e7f04dfa0410d94c5290261c065876fa5690e5f7d58daf3b6a818b21a5eee85")

	pub, _ := owcrypt.GenPubkey(prv, owcrypt.ECC_CURVE_SECP256K1)
	pub = owcrypt.PointCompress(pub, owcrypt.ECC_CURVE_SECP256K1)
	log.Infof("pub: %s", hex.EncodeToString(pub))
	decoder := AddressDecoder{}

	addr, err := decoder.PublicKeyToAddress(pub, false)
	if err != nil {
		t.Errorf("AddressDecode failed unexpected error: %v\n", err)
		return
	}
	t.Logf("addr: %s", addr)
	//	0xa8a4b2d37c591db3310df648942bf3351cecd984
}
