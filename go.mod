module github.com/blocktree/quorum-adapter

go 1.23.1

toolchain go1.23.4

require (
	github.com/astaxie/beego v1.12.1
	github.com/blocktree/go-owcrypt v1.1.14
	github.com/blocktree/openwallet/v2 v2.7.0
	github.com/ethereum/go-ethereum v1.10.17
	github.com/imroc/req v0.3.2
	github.com/shopspring/decimal v0.0.0-20200105231215-408a2507e114
	github.com/tidwall/gjson v1.9.3
)

require (
	github.com/DataDog/zstd v1.4.4 // indirect
	github.com/Sereal/Sereal v0.0.0-20200210135736-180ff2394e8a // indirect
	github.com/StackExchange/wmi v0.0.0-20180116203802-5d049714c4a6 // indirect
	github.com/asdine/storm v2.1.2+incompatible // indirect
	github.com/awnumar/memcall v0.4.0 // indirect
	github.com/awnumar/memguard v0.22.5 // indirect
	github.com/blocktree/go-owcdrivers v1.2.0 // indirect
	github.com/btcsuite/btcd v0.23.1 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.1.3 // indirect
	github.com/btcsuite/btcd/btcutil v1.1.0 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.0.1 // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/drand/kyber v1.1.4 // indirect
	github.com/go-ole/go-ole v1.2.1 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/phoreproject/bls v0.0.0-20200525203911-a88a5ae26844 // indirect
	github.com/rjeczalik/notify v0.9.1 // indirect
	github.com/shiena/ansicolor v0.0.0-20151119151921-a422bbe96644 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tklauser/go-sysconf v0.3.5 // indirect
	github.com/tklauser/numcpus v0.2.2 // indirect
	go.etcd.io/bbolt v1.3.3 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
)

//replace github.com/blocktree/openwallet/v2 => ../openwallet
//
//replace github.com/blocktree/go-owcdrivers => ../go-owcdrivers
//
//replace github.com/blocktree/go-owcrypt => ../go-owcrypt
