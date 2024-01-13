# quorum-adapter

本项目适配了openwallet.AssetsAdapter接口，给应用提供了底层的区块链协议支持。

## 如何测试

openwtester包下的测试用例已经集成了openwallet钱包体系，创建conf文件，新建QUORUM.ini文件，编辑如下内容：

```ini

#full node rpc
ServerAPI = "http://127.0.0.1:10001"
# fix gas limit
fixGasLimit = ""
# Cache data file directory, default = "", current directory: ./data
dataDir = ""
# fix gas price
fixGasPrice = ""
# nonce compute mode, 0: auto-increment nonce, 1: latest nonce
nonceComputeMode = 0
# Use QuickNode Single Flight RPC
useQNSingleFlightRPC = 1
# Detect unknown contracts
detectUnknownContracts = 0
# moralis API Key
moralisAPIKey = ""
# moralis API chain
moralisAPIChain = "eth"
# use moralis API parse Block
useMoralisAPIParseBlock = 0
```
