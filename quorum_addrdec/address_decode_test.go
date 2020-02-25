package quorum_addrdec

import (
	"encoding/hex"
	"testing"
)

func TestAddressDecoder_AddressEncode(t *testing.T) {
	Default.IsTestNet = false

	pub, _ := hex.DecodeString("032144da84e7c0037014be1332617ceec15d3561dc209a1d984bf74677a41a63d0")
	addr, _ := Default.AddressEncode(pub)
	t.Logf("addr: %s", addr)
	//	0x5f75ef82839fdc491f15816fce5184f9b65fe0f8
}

func TestAddressDecoder_AddressDecode(t *testing.T) {

	Default.IsTestNet = false

	addr := "0x5f75ef82839fdc491f15816fce5184f9b65fe0f8"
	hash, _ := Default.AddressDecode(addr)
	t.Logf("hash: %s", hex.EncodeToString(hash))
}
