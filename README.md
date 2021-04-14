# Touchstone

Touchstone is a tool to verify Badge transactions. It brings true decentralization to BSV tokens.
试金石是一个用来验证 Badge 交易的工具。它个体 BSV 上的通证实现了真正的去中心化。

## <span id="index">Index</span>

- [index](#index)
- [introduction](#introduce)
- [how to build](#howtobuild)
- [how to run](#howtorun)
- [api methods](#apimethods)

## <span id="introduction">Introduction</span>

_Who can use touchstone?_
_谁能使用试金石？_

Wallets that want to support badges or anyone with a server that wants to verify Badge transactions.
想支持 Badge 的钱包或者交易所，或者想验证自己收到的 Badge 的个人。

_What problem does it solve?_
_试金石解决了什么问题？_

Before Touchstone, if a wallet user received a transaction that included token information (often in OP_RETURN), it was impossible to verify without asking a trusted 3rd-party oracle. That is centralized and inefficient.
有了试金石之前，如果一个钱包收到了一笔附带 Badge 信息的交易（一般是在 OP_RETURN 附带）那么他很难判断这个 token 是不是合法的。之前的解决方法是访问一个中心化的 oracle（预言节点）服务器，那样不去中心化而且效率很低。

Touchstone can verify a token without a third party oracle. Anyone who wants to verify Badges can run their own server and do so. Because verifying the token does not need to rely on 3rd party, this saves transaction time and cost.
试金石可以自己验证通证，不用依赖第三方 oracle，任何有运行服务器的人可以验证 Badge 通证交易。因为不用依赖第三方，能节省时间和交易成本。

- [background](#background)
- [badge script](#badgescript)
- [badge protocol](#badgeprotocol)

### <span id="background">Background</span>

There are currently several methods in the BSV ecosystem to create tokens. In most of those methods, token information is stored inside BSV transactions (layer-1). For example, the OP_RETURN might include data saying essentially "this transaction contains 50 tokens". This information is publicly auditable.

However, when receiving such a transaction, how am I to know that these are valid tokens coming from an address that actually owned the tokens, and they haven’t been sent to someone else as well (i.e. double-spent)? This validation process usually occurs off-chain (layer-2), and most methods now currently rely on a centralized server to do so.

_Badge_ is a token issuance method, also based on the layer-1/layer-2 model, but does not require a centralized validator. It is based on the Bitcoin utxo model, which enables decentralized transfer and validation of tokens.

_Touchstone_ is a tool to verify badge transactions. It is designed to be run on a server, and servers running Touchstone can be called Touchstone nodes.

While a purely layer-1 solution to token issuance has yet to be found, we can now, with Badge and Touchstone, at least make the layer-2 validation more decentralized and efficient. Wallet providers can confidently accept tokens created by users of other wallets, and quickly verify them on their own server. This saves time and bandwidth by removing the need for several external requests, and also increases trust and confidence in the tokens.

BSV 生态系统中目前有几种创建 token 数字通证的方法。在大多数这些方法中，token 信息存储在 BSV 交易中（第 1 层）。例如，OP_RETURN 可能包含一个消息，本质上在说"此交易包含 50 个 token"。此信息是可公开审核的。但是，当我收到这样的交易时，我怎么知道这些是来自实际拥有这些 token 的地址的合法 token，并且他有没有发送给其他人（双花）？该验证过程通常发生在链下（第 2 层），并且当前大多数方法都依赖中心化服务器来执行。

*Badge*是一种 token 发行方法，也基于第 1 层/第 2 层模型，但是不需要中心化验证。它基于比特币 utxo 模型，该模型可实现 token 的去中心化转移和验证。

*Touchstone/试金石*是验证 Badge 交易的工具。它是在服务器上运行，运行 Touchstone 的服务器可以称为 Touchstone 节点。

尽管世上还未实现纯粹的第 1 层 token 发行解决方案，但是我们现在可以使用 Badge 和 Touchstone 至少能使第 2 层验证更去中心化和高效。钱包提供商可以放心地接受其他钱包用户创建的 token，并在自己的服务器上快速对其进行验证。能减少外部请求、节省了时间和网费，并且还增加了对 token 的信任和信心。

### <span id="badgescript">Badge Script</span>

#### Badge vout

Badges is a script written in [sCrypt](https://scrypt.io/):

```c#
contract Badge{
    Ripemd160 pubKeyHash;

    constructor(Ripemd160 pubKeyHash){
        this.pubKeyHash = pubKeyHash;
    }

    public function unlock(Sig sig,PubKey pubKey,bytes badgeFlag){
        require(badgeFlag == b'6261646765');
        require(hash160(pubKey) == this.pubKeyHash);
        require(checkSig(sig,pubKey));
    }
}
```

When badges are issued(minted) or transferred, they are included in a 'lock script' within a Bitcoin transaction. The transaction `vout` will look like this:

```
1 64 0 81 -49 -50 OP_NOP <pubkey hash> 0 1 OP_PICK 1 OP_ROLL OP_DROP OP_NOP 8 OP_PICK 6261646765 OP_EQUAL OP_VERIFY 9 OP_PICK OP_HASH160 1 OP_PICK OP_EQUAL OP_VERIFY 10 OP_PICK 10 OP_PICK OP_CHECKSIG OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_RETURN 1027000000000000
```

This is saying that public key: `<pubkey hash>` owns: `10000`, (or 1027000000000000 in [Little-Endian](https://learnmeabitcoin.com/technical/little-endian)), of some kind of badge

A badge‘s ID is the first transaction's txid
Often times a description and some metadata about the token will also be included in the first (minting) transactino OP_RETURN

#### Badge vin

To unlock the coins, the `vout` above, you will need to create a `vin` (Written in [p2pkh](https://learnmeabitcoin.com/technical/p2pkh)) like:

```bash
# For `<badge-flag>` pass in "badge" .
<sig> <pubkey> <badge-flag>
```

### <span id="badgeprotocol">Badge Protocol</span>

Consider we have a badge utxo set,let see what will happen when transtions come. For now ,the utxo set is empty.

- issuance

If there is a `tx1` like this:

| vin           | vout                     |
| ------------- | ------------------------ |
| not badge vin | badge vout 1, contain 50 |
| ...           | badge vout 2, contain 40 |
|               | badge vout 3, contain 30 |
|               | badge vout 4, contain 30 |
|               | ...                      |

with at least one badge format vout and no badge format vin,we consider it is a badge issuance. Vout order does not matter. It can also contain other kind srcipt.

And now ,our utxo set got

`<badge vout 1,contain 50>` `<badge vout 2,contain 40>` `<badge vout 3,contain 30>` `<badge vout 4,contain 30>` , the transaction issue 150 badge.

And another `tx2` like:

| vin                  | vout                              |
| -------------------- | --------------------------------- |
| not badge format vin | badge format vout 5, contain 1000 |
| ...                  | ...                               |

our utxo set will be:

`<badge vout 1,contain 50>` `<badge vout 2,contain 40>` `<badge vout 3,contain 30>` `<badge vout 4,contain 30>` `<badge vout 5,contain 1000>`.

But we should know `<badge format vout 5,contain 1000>>` is different from others. May be we can call them

`<badge vout 1,contain 50 of b1>` `<badge vout 2,contain 40 of b1>` `<badge vout 3,contain 30 of b1>` `<badge vout 4,contain 30 of b1>` `<badge vout 5,contain 1000 of b2>`

- transfer

If alice wants to transfer bob 70 b1,carol 10 b1,and get change, we may see `tx3` like:

| vin                             | vout                            |
| ------------------------------- | ------------------------------- |
| <badge vout 1,contain 50 of b1> | <badge vout 6,contain 70 of b1> |
| <badge vout 2,contain 40 of b1> | <badge vout 7,contain 10 of b1> |
| ...                             | <badge vout 8,contain 10 of b1> |

our utxo set will be:

`<badge vout 3,contain 30 of b1>` `<badge vout 4,contain 30 of b1>` `<badge vout 5,contain 1000 of b2>` `<badge vout 6,contain 70 of b1>` `<badge vout 7,contain 10 of b1>` `<badge vout 8,contain 10 of b1>`

If we see `tx4`

| vin                             | vout                            |
| ------------------------------- | ------------------------------- |
| <badge vout 3,contain 30 of b1> | <badge vout 9,contain 41 of b1> |
| <badge vout 7,contain 10 of b1> | ...                             |
| ...                             |                                 |

because vout badge count is greater than vin,so `<badge vout 9,contain 41 of b1>` will not be token,but `<badge vout 3,contain 30 of b1>` `<badge vout 7,contain 20 of b1>` is still gone.

our utxo set will be:

`<badge vout 4,contain 30 of b1>` `<badge vout 5,contain 1000 of b2>` `<badge vout 6,contain 70 of b1>` `<badge vout 8,contain 10 of b1>`

If we see tx5

| vin                               | vout                              |
| --------------------------------- | --------------------------------- |
| <badge vout 4,contain 30 of b1>   | <badge vout 10,contain 1030 of ?> |
| <badge vout 5,contain 1000 of b2> | ...                               |
| ...                               |                                   |

even vout badge count is not greater than vin,but vin contains `b1` and `b2`,which is not support for now. we may upgrade protocol to support this case in the future, but now ,`<badge vout 10,contain 1030 of ?>` will not be token and `<badge vout 4,contain 30 of b1>` `<badge vout 5,contain 1000 of b2>` is gone.

our utxo set will be:

`<badge vout 6,contain 70 of b1>` `<badge vout 8,contain 10 of b1>`

- burn

If we see tx6

| vin                             | vout                             |
| ------------------------------- | -------------------------------- |
| <badge vout 6,contain 70 of b1> | <badge vout 11,contain 20 of b1> |
| ...                             | ...                              |

our utxo set will be:  
`<badge vout 8,contain 10 of b1>` `<badge vout 11,contain 20 of b1>`

and tx7 :

| vin                             | vout |
| ------------------------------- | ---- |
| <badge vout 8,contain 10 of b1> | ...  |
| ...                             | ...  |

our utxo set will be:  
`<badge vout 11,contain 20 of b1>`

If there is a transaction,vout badge count is less than vin,we can easily find that total badge utxo set contain is getting less. So we can use `tx6` and `tx7` to burn the badge. By the way, `tx4` and `tx5` also reduce the count of badge, you may use it too.

## <span id="howtobuild">How To Build</span>

- Make sure you got golang and all grpc protobuf support in your env

```shell
sh build.sh
```

## <span id="howtorun">How To Run</span>

To run a touchstone node , you at least need

- mongo
- mapi support
- a private key of bitcoin for yourself
- known peers pubkey and host
- let peers have your pubkey

Here is a example for config

```json
{
	"Env": "mainnet",
	"MongoHost": "127.0.0.1:27017",
	"MempoolHost": "https://api.ddpurse.com",
	"MempoolPkiMnemonic": "border napkin domain blush hammer what avocado venue delay network tell art",
	"MempoolPkiMnemonicPassword": "",
	"ServerPrivatekey": "9a4d8f5f2f7ad34f90bfcafe2961aabc71bdee0df63f3c4cc2b95fbc93a5572f",
	"PeersConfigs": [
		{
			"Host": "127.0.0.1:7788",
			"Pubkey": "036af584f4f274e3b6831f9c8cfb8cce56d441887a9349cc93b180eb9a913d06cd"
		},
		{
			"Host": "127.0.0.1:7788",
			"Pubkey": "026e85c3255ad46183ee3e425247e4a326be03c4ea5f2be5f8d6280610e0376492"
		}
	],
	"P2pHost": "0.0.0.0:7788",
	"HttpHost": "0.0.0.0:7789",
	"DbName": "touchstone"
}
```

and then just run

```shell
./touchstone -config=config.json -log_dir=logs
```

### mapi support

- This version of code only support mapi provided by mempool, you can easily replace it by any provider. Just implement `MapiClientAdaptor` in `mapi/mapi_client.go`,and modify code in `main.go`

```golang
	mapiClient, err := mapi.NewMempoolMapiClient(config.MempoolHost, config.MempoolPkiMnemonic, config.MempoolPkiMnemonicPassword)
```

For now ,touchstone relay on mapi to verify transaction.In the future, we may access bitcoin p2p network

## <span id="apimethod">Api Method</span>

- [sendrawtransaction](#sendrawtransaction)

- [gettxinventory](#gettxinventory)

- [getaddrutxos](#getaddrutxos)

- [getaddrbalance](#getaddrbalance)

- [getaddrinventorys](#getaddrinventorys)

- [setaddrinfo](#setaddrinfo)

- [getuserutxos](#getuserutxos)

- [getuserbalance](#getuserbalance)

- [getuserinventorys](#getuserinventorys)

- [sendbadgetoaddress](#sendbadgetoaddress)

### <span id="sendrawtransaction">sendrawtransaction</span>

- params

| param | required | note            |
| ----- | -------- | --------------- |
| rawtx | true     | raw transaction |

- req

```shell
curl -X POST --data '{
    "rawtx":"0200000002eccdb8ab6330d13916c1ef0c3770e99824a7ffb8a81a73d0ea0432e18dbd437d09000000704730440220119a76cc371b08f30846490477dc1a3b78a44975efe70595f57d5696efb9d24d02206a329e11a2f5ddaed6c1247ba1dbb3098689ba2ba5bc9a49e83d9d6876d13dc941210285505a06efbdfbcd0761bb6ac3a1ff077ad9cfd5a1866e3acfc965a4a7a71b5d056261646765ffffffff3d0978d82c653db53a2cdb77031fadcd81033ddba3ac628fc141a57b1706e4e1020000006a473044022009a10596ae7678fbd7fef19f9252160726c3caa46d19b2e812a945114388779c0220103d55b8d22bcdabbc671c02a3cdd45fb3f244c076b89cc9569e301585117dd4412102cfc43c675370c3f0050ffc1d340493fc74e6ca8bb6280c27fd336372f41b2b20ffffffff047803000000000000535101400100015101b101b26114ebaa76479d676a18e9967b09d4939bc90c4b3246005179517a7561587905626164676587695979a9517987695a795a79ac77777777777777777777776a0820a10700000000007803000000000000535101400100015101b101b2611474b0bbf85657c5c3e412fcf003ee713ade38c2a4005179517a7561587905626164676587695979a9517987695a795a79ac77777777777777777777776a0820aa4400000000007803000000000000535101400100015101b101b2611452dee27ceab2761f3e72176b2dfad30f0f03bacd005179517a7561587905626164676587695979a9517987695a795a79ac77777777777777777777776a083cc43a000000000069713600000000001976a91452dee27ceab2761f3e72176b2dfad30f0f03bacd88ac00000000"
}' http://127.0.0.1:7789/v1/touchstone/sendrawtransaction
```

- rsp

  - when it came to vout,`pretxid` and `preindex` will always be empty str and -1

```json
{
	"code": 0,
	"msg": "",
	"data": {
		"vins": [
			{
				"addr": "155ruNknRz9ZHcnTgsW5KkzePZoE91RMee",
				"txid": "443110747ccd782c7dc480daec7faee63533c4c2cdf7231966d7a0cc61036132",
				"index": 0,
				"value": -8851324,
				"pretxid": "7d43bd8de13204ead0731aa8b8ffa72498e970370cefc11639d13063abb8cdec",
				"preindex": 9,
				"badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615197182
			}
		],
		"vouts": [
			{
				"addr": "1NV5zzQBJ7v5DZtDS6CewnvkvHkVHsTnPR",
				"txid": "443110747ccd782c7dc480daec7faee63533c4c2cdf7231966d7a0cc61036132",
				"index": 0,
				"value": 500000,
				"pretxid": "",
				"preindex": -1,
				"badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615197182
			},
			{
				"addr": "1Be16xuRBoK91LdbbZidSJm5frpsX3LJ6J",
				"txid": "443110747ccd782c7dc480daec7faee63533c4c2cdf7231966d7a0cc61036132",
				"index": 1,
				"value": 4500000,
				"pretxid": "",
				"preindex": -1,
				"badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615197182
			},
			{
				"addr": "18ZBR2r6GcBW5ZsNsWDphbYB48Xw2FctaM",
				"txid": "443110747ccd782c7dc480daec7faee63533c4c2cdf7231966d7a0cc61036132",
				"index": 2,
				"value": 3851324,
				"pretxid": "",
				"preindex": -1,
				"badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615197182
			}
		]
	}
}
```

### <span id="gettxinventory">gettxinventory</span>

- params

| param | required | note           |
| ----- | -------- | -------------- |
| txid  | true     | transaction id |

- req

```shell
curl -X POST --data '{
    "txid":"443110747ccd782c7dc480daec7faee63533c4c2cdf7231966d7a0cc61036132"
}' http://127.0.0.1:7789/v1/touchstone/gettxinventory
```

- rsp
  - when it came to vout,`pretxid` and `preindex` will always be empty string and -1

```json
{
	"code": 0,
	"msg": "",
	"data": {
		"vins": [
			{
				"addr": "155ruNknRz9ZHcnTgsW5KkzePZoE91RMee",
				"txid": "443110747ccd782c7dc480daec7faee63533c4c2cdf7231966d7a0cc61036132",
				"index": 0,
				"value": -8851324,
				"pretxid": "7d43bd8de13204ead0731aa8b8ffa72498e970370cefc11639d13063abb8cdec",
				"preindex": 9,
				"badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615197182
			}
		],
		"vouts": [
			{
				"addr": "1NV5zzQBJ7v5DZtDS6CewnvkvHkVHsTnPR",
				"txid": "443110747ccd782c7dc480daec7faee63533c4c2cdf7231966d7a0cc61036132",
				"index": 0,
				"value": 500000,
				"pretxid": "",
				"preindex": -1,
				"badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615197182
			},
			{
				"addr": "1Be16xuRBoK91LdbbZidSJm5frpsX3LJ6J",
				"txid": "443110747ccd782c7dc480daec7faee63533c4c2cdf7231966d7a0cc61036132",
				"index": 1,
				"value": 4500000,
				"pretxid": "",
				"preindex": -1,
				"badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615197182
			},
			{
				"addr": "18ZBR2r6GcBW5ZsNsWDphbYB48Xw2FctaM",
				"txid": "443110747ccd782c7dc480daec7faee63533c4c2cdf7231966d7a0cc61036132",
				"index": 2,
				"value": 3851324,
				"pretxid": "",
				"preindex": -1,
				"badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615197182
			}
		]
	}
}
```

### <span id="getaddrutxos">getaddrutxos</span>

- params

| param      | required | note             |
| ---------- | -------- | ---------------- |
| addr       | true     | addr             |
| badge_code | true     | badge code       |
| offset     | false    | offset,defalut 0 |
| limit      | false    | limit,defalut 10 |

- req

```shell
curl -X POST --data '{
    "badge_code":"e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
    "addr":"1LRKoKfHef3DMZ7aLqAiwsf1a3TQYQ4G9i"
}' http://127.0.0.1:7789/v1/touchstone/getaddrutxos
```

- rsp
  - when it came to vout,`pretxid` and `preindex` will always be empty string and -1

```json
{
	"code": 0,
	"msg": "",
	"data": {
		"utxos": [
			{
				"addr": "1LRKoKfHef3DMZ7aLqAiwsf1a3TQYQ4G9i",
				"txid": "7d43bd8de13204ead0731aa8b8ffa72498e970370cefc11639d13063abb8cdec",
				"index": 0,
				"value": 8976,
				"pretxid": "",
				"preindex": -1,
				"badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615196949
			},
			{
				"addr": "1LRKoKfHef3DMZ7aLqAiwsf1a3TQYQ4G9i",
				"txid": "7d43bd8de13204ead0731aa8b8ffa72498e970370cefc11639d13063abb8cdec",
				"index": 1,
				"value": 37778,
				"pretxid": "",
				"preindex": -1,
				"badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615196949
			}
		]
	}
}
```

### <span id="getaddrbalance">getaddrbalance</span>

- params

| param      | required | note       |
| ---------- | -------- | ---------- |
| addr       | true     | addr       |
| badge_code | true     | badge code |

- req

```shell
curl -X POST --data '{
    "badge_code":"e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
    "addr":"1LRKoKfHef3DMZ7aLqAiwsf1a3TQYQ4G9i"
}' http://127.0.0.1:7789/v1/touchstone/getaddrbalance
```

- rsp

```json
{
	"code": 0,
	"msg": "",
	"data": {
		"balance": 46754
	}
}
```

### <span id="getaddrinventorys">getaddrinventorys</span>

- params

| param      | required | note             |
| ---------- | -------- | ---------------- |
| addr       | true     | addr             |
| badge_code | true     | badge code       |
| offset     | false    | offset,defalut 0 |
| limit      | false    | limit,defalut 10 |

- req

```shell
curl -X POST --data '{
    "badge_code":"e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
    "addr":"1PLuQQPRBcpDurPc9bZAw5pcePgCNatCfG"
}' http://127.0.0.1:7789/v1/touchstone/getaddrinventorys
```

- rsp

```json
{
	"code": 0,
	"msg": "",
	"data": {
		"addr_inventorys": [
			{
				"addr": "1PLuQQPRBcpDurPc9bZAw5pcePgCNatCfG",
				"txid": "7d43bd8de13204ead0731aa8b8ffa72498e970370cefc11639d13063abb8cdec",
				"timestamp": 1615196949,
				"value": -1000000000000
			},
			{
				"addr": "1PLuQQPRBcpDurPc9bZAw5pcePgCNatCfG",
				"txid": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615196586,
				"value": 1000000000000
			}
		]
	}
}
```

### <span id="setaddrinfo">setaddrinfo</span>

- params

| param      | required | note                                                  |
| ---------- | -------- | ----------------------------------------------------- |
| userid     | true     | user id                                               |
| appid      | true     | recommended to indicate the usefulness of the address |
| user_index | true     | in case one user have more than one wallet            |
| addr       | true     | addr                                                  |

- req

```shell
curl -X POST --data '{
    "userid":1,
    "appid":"auto pay",
    "user_index":0,
    "addr":"1PLuQQPRBcpDurPc9bZAw5pcePgCNatCfG"
}' http://127.0.0.1:7789/v1/touchstone/setaddrinfo
```

- rsp

```json
{
	"code": 0,
	"msg": "",
	"data": null
}
```

### <span id="getuserutxos">getuserutxos</span>

- params

| param      | required | note                          |
| ---------- | -------- | ----------------------------- |
| appid      | true     | app id set by setaddrinfo     |
| userid     | true     | user id set by setaddrinfo    |
| user_index | true     | user index set by setaddrinfo |
| badge_code | true     | badge code                    |
| offset     | false    | offset,defalut 0              |
| limit      | false    | limit,defalut 10              |

- req

```shell
curl -X POST --data '{
    "userid":1,
    "appid":"auto pay",
    "user_index":1,
    "badge_code":"e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700"
}' http://127.0.0.1:7789/v1/touchstone/getuserutxos
```

- rsp

```json
{
	"code": 0,
	"msg": "",
	"data": {
		"utxos": [
			{
				"addr": "1LRKoKfHef3DMZ7aLqAiwsf1a3TQYQ4G9i",
				"txid": "7d43bd8de13204ead0731aa8b8ffa72498e970370cefc11639d13063abb8cdec",
				"index": 0,
				"value": 8976,
				"pretxid": "",
				"preindex": -1,
				"badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615270358
			},
			{
				"addr": "1LRKoKfHef3DMZ7aLqAiwsf1a3TQYQ4G9i",
				"txid": "7d43bd8de13204ead0731aa8b8ffa72498e970370cefc11639d13063abb8cdec",
				"index": 1,
				"value": 37778,
				"pretxid": "",
				"preindex": -1,
				"badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615270358
			}
		]
	}
}
```

### <span id="getuserbalance">getuserbalance</span>

- params

| param      | required | note                          |
| ---------- | -------- | ----------------------------- |
| appid      | true     | app id set by setaddrinfo     |
| userid     | true     | user id set by setaddrinfo    |
| user_index | true     | user index set by setaddrinfo |
| badge_code | true     | badge code                    |

- req

```shell
curl -X POST --data '{
    "userid":1,
    "appid":"auto pay",
    "user_index":1,
    "badge_code":"e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700"
}' http://127.0.0.1:7789/v1/touchstone/getuserbalance
```

- rsp

```json
{
	"code": 0,
	"msg": "",
	"data": {
		"balance": 46754
	}
}
```

### <span id="getuserinventorys">getuserinventorys</span>

- params

| param      | required | note                          |
| ---------- | -------- | ----------------------------- |
| appid      | true     | app id set by setaddrinfo     |
| userid     | true     | user id set by setaddrinfo    |
| user_index | true     | user index set by setaddrinfo |
| badge_code | true     | badge code                    |
| offset     | false    | offset,defalut 0              |
| limit      | false    | limit,defalut 10              |

- req

```shell
curl -X POST --data '{
    "userid":1,
    "appid":"auto pay",
    "user_index":1,
    "badge_code":"e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700"
}' http://127.0.0.1:7789/v1/touchstone/getuserinventorys
```

- rsp

```json
{
	"code": 0,
	"msg": "",
	"data": {
		"addr_inventorys": [
			{
				"addr": "1PLuQQPRBcpDurPc9bZAw5pcePgCNatCfG",
				"txid": "7d43bd8de13204ead0731aa8b8ffa72498e970370cefc11639d13063abb8cdec",
				"timestamp": 1615270358,
				"value": -999999953246
			},
			{
				"addr": "1PLuQQPRBcpDurPc9bZAw5pcePgCNatCfG",
				"txid": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615268767,
				"value": 1000000000000
			}
		]
	}
}
```

### <span id="sendbadgetoaddress">sendbadgetoaddress</span>

- describe

create a unsigned insufficient fee transaction

- params

| param        | required | note                                                                 |
| ------------ | -------- | -------------------------------------------------------------------- |
| appid        | true     | app id set by setaddrinfo                                            |
| userid       | true     | user id set by setaddrinfo                                           |
| user_index   | true     | user index set by setaddrinfo                                        |
| badge_code   | true     | badge code                                                           |
| addr_amounts | false    | reveiver's addr and amount,allow empty,for just burn or collect utxo |
| amount2burn  | false    | amount to burn                                                       |
| change_addr  | true     | change address                                                       |

- req

```shell
curl -X POST --data '{
    "userid":1,
    "appid":"auto pay",
    "user_index":1,
    "addr_amounts":[
        {
            "addr":"1DfZoSCPGsxH1JEcgViWmx72TWVAWxivpm",
            "amount":10000
        }
    ],
    "amount2burn":0,
    "badge_code":"e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
    "change_addr":"1PLuQQPRBcpDurPc9bZAw5pcePgCNatCfG"
}' http://127.0.0.1:7789/v1/touchstone/sendbadgetoaddress
```

- rsp

```json
{
	"code": 0,
	"msg": "",
	"data": {
		"unfinished_tx": "0200000002eccdb8ab6330d13916c1ef0c3770e99824a7ffb8a81a73d0ea0432e18dbd437d0000000000ffffffffeccdb8ab6330d13916c1ef0c3770e99824a7ffb8a81a73d0ea0432e18dbd437d0100000000ffffffff027803000000000000535101400100015101b101b261148aecb24e78012e629bb3bbd3f834eaeeb41f2330005179517a7561587905626164676587695979a9517987695a795a79ac77777777777777777777776a0810270000000000007803000000000000535101400100015101b101b26114f5166a2da65a519a1620242ac793e92dcfc62652005179517a7561587905626164676587695979a9517987695a795a79ac77777777777777777777776a08928f00000000000000000000",
		"vins": [
			{
				"addr": "1LRKoKfHef3DMZ7aLqAiwsf1a3TQYQ4G9i",
				"txid": "7d43bd8de13204ead0731aa8b8ffa72498e970370cefc11639d13063abb8cdec",
				"index": 0,
				"value": 8976,
				"pretxid": "",
				"preindex": -1,
				"badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615270358
			},
			{
				"addr": "1LRKoKfHef3DMZ7aLqAiwsf1a3TQYQ4G9i",
				"txid": "7d43bd8de13204ead0731aa8b8ffa72498e970370cefc11639d13063abb8cdec",
				"index": 1,
				"value": 37778,
				"pretxid": "",
				"preindex": -1,
				"badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
				"timestamp": 1615270358
			}
		]
	}
}
```
