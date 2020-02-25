package quorum

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/blocktree/openwallet/common"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/status-im/keycard-go/hexutils"
	"math/big"
	"strings"
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

//调用合约ABI方法
func (decoder *EthContractDecoder) CallSmartContractABI(wrapper openwallet.WalletDAI, rawTx *openwallet.SmartContractRawTransaction) (*openwallet.SmartContractCallResult, *openwallet.Error) {

	var (
		callMsg CallMsg
		err     error
	)

	abiInstance, err := abi.JSON(strings.NewReader(rawTx.Coin.Contract.GetABI()))
	if err != nil {
		return nil, openwallet.ConvertError(err)
	}

	if len(rawTx.Raw) > 0 {
		//直接的数据请求
		switch rawTx.RawType {
		case openwallet.TxRawTypeHex:

			rawBytes := hexutils.HexToBytes(rawTx.Raw)
			err = rlp.DecodeBytes(rawBytes, &callMsg)
			if err != nil {
				return nil, openwallet.ConvertError(err)
			}

		case openwallet.TxRawTypeJSON:
			err = json.Unmarshal([]byte(rawTx.Raw), callMsg)
			if err != nil {
				return nil, openwallet.ConvertError(err)
			}
		case openwallet.TxRawTypeBase64:
			rawBytes, _ := base64.StdEncoding.DecodeString(rawTx.Raw)
			err = rlp.DecodeBytes(rawBytes, &callMsg)
			if err != nil {
				return nil, openwallet.ConvertError(err)
			}
		}
	} else {

		data, err := decoder.wm.EncodeABIParam(abiInstance, rawTx.ABIParam...)
		if err != nil {
			return nil, openwallet.ConvertError(err)
		}
		defAddress, err := wrapper.GetAssetsAccountDefAddress(rawTx.Account.AccountID)
		if err != nil {
			return nil, openwallet.ConvertError(err)
		}
		//toAddr := ethcom.HexToAddress(rawTx.Coin.Contract.Address)
		callMsg = CallMsg{
			From: defAddress.Address,
			To:   rawTx.Coin.Contract.Address,
			Data: hex.EncodeToString(data),
		}
	}

	callResult := &openwallet.SmartContractCallResult{
		Method: rawTx.ABIParam[0],
	}

	result, err := decoder.wm.EthCall(callMsg, "latest")
	if err != nil {
		callResult.Status = openwallet.SmartContractCallResultStatusFail
		callResult.Exception = err.Error()
		return callResult, openwallet.ConvertError(err)
	}

	_, rJSON, err := decoder.wm.DecodeABIResult(abiInstance, callResult.Method, result)
	if err != nil {
		return nil, openwallet.ConvertError(err)
	}

	callResult.RawHex = result
	callResult.Value = rJSON
	callResult.Status = openwallet.SmartContractCallResultStatusSuccess

	return callResult, nil
}

//创建原始交易单
func (decoder *EthContractDecoder) CreateSmartContractRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.SmartContractRawTransaction) *openwallet.Error {
	return openwallet.Errorf(openwallet.ErrSystemException, "CreateSmartContractRawTransaction not implement")
}

//SubmitRawTransaction 广播交易单
func (decoder *EthContractDecoder) SubmitSmartContractRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.SmartContractRawTransaction) (*openwallet.SmartContractReceipt, *openwallet.Error) {
	return nil, openwallet.Errorf(openwallet.ErrSystemException, "SubmitSmartContractRawTransaction not implement")
}
