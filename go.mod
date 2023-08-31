module github.com/blocktree/quorum-adapter

go 1.12

require (
	github.com/DataDog/zstd v1.4.4 // indirect
	github.com/Sereal/Sereal v0.0.0-20200210135736-180ff2394e8a // indirect
	github.com/astaxie/beego v1.12.1
	github.com/blocktree/go-owcrypt v1.1.14
	github.com/blocktree/openwallet/v2 v2.6.0
	github.com/ethereum/go-ethereum v1.10.17
	github.com/imroc/req v0.3.0
	github.com/shopspring/decimal v0.0.0-20200105231215-408a2507e114
	github.com/tidwall/gjson v1.9.3
)

//replace github.com/blocktree/openwallet/v2 => ../../openwallet
