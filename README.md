# Touchstone

Touchstone is a tool to verify Badge transactions. It brings true decentralization to BSV tokens.
试金石是一个用来验证 Badge 交易的工具。它个体 BSV 上的通证实现了真正的去中心化。

## <span id="index">Index</span>

- [Index](#index)
- [Introduction](#introduce)
- [How to build](#howtobuild)
- [How to run](#howtorun)
- [How to use](#howtouse)
- [Api methods](#apimethods)

## <span id="introduction">Introduction</span>

_Who can use touchstone?_
_谁能使用试金石？_

Wallets that want to support badges or anyone with a server that wants to verify Badge transactions.
想支持 Badge 的钱包或者交易所，或者想验证自己收到的 Badge 的个人。

_What problem does it solve?_
_试金石解决了什么问题？_

Before Touchstone, if a wallet user received a transaction that included token information (often in OP_RETURN), it was impossible to verify without asking a trusted 3rd-party oracle. That is centralized and inefficient.
有了试金石之前，如果一个钱包收到了一笔附带 Badge 信息的交易（一般是在 OP_RETURN 附带）那么他很难判断这个 token 是不是合法的。之前的解决方法是访问一个中心化的 oracle（预言节点）服务器，那样不去中心化而且效率很低。

Touchstone can verify a token without a third party oracle. Anyone who wants to verify Badges can run their own Touchstone server and do so. Because verifying the token does not need to rely on 3rd party, this saves transaction time and cost.
试金石可以自己验证通证，不用依赖第三方 oracle，任何有运行服务器的人可以验证 Badge 通证交易。因为不用依赖第三方，能节省时间和交易成本。

- [background](#background)
- [badge script](#badgescript)
- [badge protocol](#badgeprotocol)

### <span id="background">Background</span>

There are currently several methods in the BSV ecosystem to create tokens. In most of those methods, token information is stored inside BSV transactions (layer-1). For example, the OP_RETURN might include data saying essentially "this transaction contains 50 tokens". This information is publicly auditable.

However, when receiving such a transaction, how am I to know that these are valid tokens coming from an address that actually owned the tokens, and they haven’t been sent to someone else as well (i.e. double-spent)? This validation process usually occurs off-chain (layer-2), and most methods now currently rely on a centralized server to do so.

_Badge_ is a token issuance method, also based on the layer-1/layer-2 model, but does not require a centralized validator. It is based on the Bitcoin utxo model, which enables decentralized transfer and validation of tokens.

_Touchstone_ is a tool to verify badge transactions. It is designed to be run on a server, and servers running Touchstone can be called Touchstone nodes. The Touchstone node network will sync and store badge related transaction info for easy querying. The Touchstone network can also store some basic [user info](#setaddrinfo) for ease of use between wallet providers.

While a purely layer-1 solution to token issuance has yet to be found, we can now, with Badge and Touchstone, at least make the layer-2 validation more decentralized and efficient. Wallet providers can confidently accept tokens created by users of other wallets, and quickly verify them on their own server. This saves time and bandwidth by removing the need for several external requests, and also increases trust and confidence in the tokens.

BSV 生态系统中目前有几种创建 token 数字通证的方法。在大多数这些方法中，token 信息存储在 BSV 交易中（第 1 层）。例如，OP_RETURN 可能包含一个消息，本质上在说"此交易包含 50 个 token"。此信息是可公开审核的。但是，当我收到这样的交易时，我怎么知道这些是来自实际拥有这些 token 的地址的合法 token，并且他有没有发送给其他人（双花）？该验证过程通常发生在链下（第 2 层），并且当前大多数方法都依赖中心化服务器来执行。

*Badge*是一种 token 发行方法，也基于第 1 层/第 2 层模型，但是不需要中心化验证。它基于比特币 utxo 模型，该模型可实现 token 的去中心化转移和验证。

*Touchstone/试金石*是验证 Badge 交易的工具。它是在服务器上运行，运行 Touchstone 的服务器可以称为 Touchstone 节点。试金石节点网络会储存和同步 Badge 相关的交易历史和信息。试金石网络也可以储存一些和用户相关的[基本信息](#setaddrinfo)，让钱包的用户信息同步更加方便。

尽管世上还未实现纯粹的第 1 层 token 发行解决方案，但是我们现在可以使用 Badge 和 Touchstone 至少能使第 2 层验证更去中心化和高效。钱包提供商可以放心地接受其他钱包用户创建的 token，并在自己的服务器上快速对其进行验证。能减少外部请求、节省了时间和网费，并且还增加了对 token 的信任和信心。

![touchstone-flow-chart](https://gateway.pinata.cloud/ipfs/QmTpyj5H6Lrt9UgY2PQ7cC7sRFtZLdmTA86RMqCsm1TLXg)

### <span id="badgescript">Badge Script</span>

#### Badge vout

Badges is a script written in [sCrypt](https://scrypt.io/). It creates a simple token initialized with a public key, which can only be unlocked with a signature from a matching public key.

```java
// Badge contract source code (sCrypt):
contract Badge{
    Ripemd160 pubKeyHash;

    constructor(Ripemd160 pubKeyHash){
        this.pubKeyHash = pubKeyHash;
    }

    public function unlock(Sig sig, PubKey pubKey, bytes badgeFlag){
        require(badgeFlag == b'6261646765'); // b'6261646765' is "badge" in hex
        require(hash160(pubKey) == this.pubKeyHash);
        require(checkSig(sig, pubKey));
    }
}
```

When badges are issued(minted) or transferred, they are included in a 'lock script' within a Bitcoin transaction. The transaction `vout` will look like something like this:

```
1 64 0 81 -49 -50 OP_NOP <pubkey hash> 0 1 OP_PICK 1 OP_ROLL OP_DROP OP_NOP 8 OP_PICK 6261646765 OP_EQUAL OP_VERIFY 9 OP_PICK OP_HASH160 1 OP_PICK OP_EQUAL OP_VERIFY 10 OP_PICK 10 OP_PICK OP_CHECKSIG OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_RETURN 1027000000000000
```

This is saying that public key: `<pubkey hash>` owns: `10000`, (or 1027000000000000 in [Little-Endian](https://learnmeabitcoin.com/technical/little-endian)), of some kind of badge

A badge‘s ID is the first transaction's txid.
A description and some metadata about the token will also often be included in the first (minting) transaction's OP_RETURN

#### Badge vin

To unlock the tokens from the `vout` above, you will need to create a `vin` (Written in [p2pkh](https://learnmeabitcoin.com/technical/p2pkh)) like:

```bash
# For `<badge-flag>` pass in "badge" .
<sig> <pubkey> <badge-flag>
# Example:
304402200...40851a70cb02 026f...aee8b78b badge
```

### <span id="badgeprotocol">Badge Protocol</span>

#### Issuance

If the utxo set has at least one badge format vout and no badge format vin, it is a badge issuance.
during issuance. there can be no badge vins. The badge code will be the issuance transaction's txid

An initial issuing, or 'minting', of badge tokens will have a utxo set like this:

| txid: 0001 | vin                 | vout                                |
| ---------- | ------------------- | ----------------------------------- |
|            | no badge format vin | badge vout 1: 1000 tokens of tx0001 |
|            | ...                 | ...                                 |

Which shows that this transaction is issuing 1000 tokens.

Issuances can also have multiple vouts:

| txid: 0002 | vin                 | vout                              |
| ---------- | ------------------- | --------------------------------- |
|            | no badge format vin | badge vout 1: 90 tokens of tx0002 |
|            | ...                 | badge vout 2: 10 tokens of tx0002 |
|            |                     | badge vout 3: 80 tokens of tx0002 |
|            |                     | badge vout 4: 20 tokens of tx0002 |
|            |                     | ...                               |

Which shows that this transaction is issuing 200 tokens split between 4 outputs

> Vout order does not matter. Vouts can also contain other kinds of scripts without effecting the badges.

> You can not issue two kinds of badges in one transaction.

### Transfer

Say Alice received 100 of the tx0002 tokens (vout 1 and 2 above). If she wants to send Bob 70 tx0002 tokens, Carol 10, and get back the change, we could create a utxo set like this:

| txid: 0003 | vin                                             | vout                              |
| ---------- | ----------------------------------------------- | --------------------------------- |
|            | badge vout 1 (from tx0002): 90 tokens of tx0002 | badge vout 1: 70 tokens of tx0002 |
|            | badge vout 2 (from tx0002): 10 tokens of tx0002 | badge vout 2: 10 tokens of tx0002 |
|            |                                                 | badge vout 3: 20 tokens of tx0002 |

Bob could then send his tokens to Dave and Edward:

| txid: 0004 | vin                                             | vout                              |
| ---------- | ----------------------------------------------- | --------------------------------- |
|            | badge vout 1 (from tx0003): 70 tokens of tx0002 | badge vout 1: 50 tokens of tx0002 |
|            |                                                 | badge vout 2: 20 tokens of tx0002 |

#### Invalid transfer: vout > vin

Transfers where the sum of tokens in vout exceeds the sum of tokens vin are invalid:

| txid: 0005 | vin                     | vout                    |
| ---------- | ----------------------- | ----------------------- |
|            | badge vout 1: 90 tokens | badge vout 1: 99 tokens |

> These transfers are still recorded on chain. The tokens from the vin of an invalid transfer will be burned.

#### Invalid transfer: 2 different badge tokens

| txid: 0006 | vin                                              | vout                              |
| ---------- | ------------------------------------------------ | --------------------------------- |
|            | badge vout 1 (from tx0001): 100 tokens of tx0001 | badge vout 1: 50 tokens of tx0001 |
|            | badge vout 2 (from tx0002): 70 tokens of tx0002  | badge vout 1: 50 tokens of tx0002 |

We may upgrade protocol to support this case in the future, but now for now, this kind of transaction will be invalid.

> The tokens from the vin of an invalid transfer will be burned.

### Burning tokens

If the sum of the tokens in vout is less than the sum of the tokens in vin, the remaining tokens will be burned.

| txid: 0007 | vin                      | vout                    |
| ---------- | ------------------------ | ----------------------- |
|            | badge vout 1: 100 tokens | badge vout 1: 50 tokens |
|            |                          | badge vout 2: 10 tokens |

Because vin contains 100 tokens and vout contains 60 tokens, 40 tokens were burned.

Transactions with empty vouts also burn tokens:

| txid: 0007 | vin                      | vout |
| ---------- | ------------------------ | ---- |
|            | badge vout 1: 100 tokens |      |

All 100 tokens were burned

## <span id="howtobuild">How To Build</span>

- Make sure you have golang and all grpc protobuf supported in your env

```shell
sh build.sh
```

## <span id="howtorun">How To Run</span>

To run a touchstone node, you need:

To run a touchstone node, you'll need:

- Mongo DB
- MAPI support
- A Bitcoin private key that belongs to you
- Public keys and ip ports of other nodes
- To let other nodes know your public key

Example configuration:

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

Then just run:

```shell
./touchstone -config=config.json -log_dir=logs
```

### Mapi support

- This version of Touchstone only supports mapi provided by mempool, but you can edit the code to easily replace it with any provider. Just implement `MapiClientAdaptor` in `mapi/mapi_client.go`, and modify the code in `main.go`

```golang
	mapiClient, err := mapi.NewMempoolMapiClient(config.MempoolHost, config.MempoolPkiMnemonic, config.MempoolPkiMnemonicPassword)
```

For now, touchstone relies on mapi to verify transactions. In the future, we may change it to access the bitcoin p2p network directly.

## <span id="howtouse">How To Use</span>

1. Send tokens:

   - use [sendbadgetoaddress](#sendbadgetoaddress) to create an unsigned transaction.
   - Sign it and send it like any other Bitcoin SV transaction.

2. Verify badge transactions:
   - Most of the rest of the API methods can be used to check how many valid badge tokens were included in a transaction or are associated with an address, and to query badge transaction histories.
3. Save and get user info:
   - Use [setaddrinfo](#setaddrinfo) and [getuserbalance](#getuserbalance)

## <span id="apimethod">Api Methods</span>

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

- Description

Broadcast a transaction to the Touchstone nodes network. Other touchstone nodes will save and sync this transaction. This must be a completed raw transaction from a successful on-chain transaction.

- Params

| param | required | note            |
| ----- | -------- | --------------- |
| rawtx | true     | raw transaction |

- Request

```shell
curl -X POST --data '{
    "rawtx":"0200000002eccdb8ab6330d13916c1ef0c3770e99824a7ffb8a81a73d0ea0432e18dbd437d09000000704730440220119a76cc371b08f30846490477dc1a3b78a44975efe70595f57d5696efb9d24d02206a329e11a2f5ddaed6c1247ba1dbb3098689ba2ba5bc9a49e83d9d6876d13dc941210285505a06efbdfbcd0761bb6ac3a1ff077ad9cfd5a1866e3acfc965a4a7a71b5d056261646765ffffffff3d0978d82c653db53a2cdb77031fadcd81033ddba3ac628fc141a57b1706e4e1020000006a473044022009a10596ae7678fbd7fef19f9252160726c3caa46d19b2e812a945114388779c0220103d55b8d22bcdabbc671c02a3cdd45fb3f244c076b89cc9569e301585117dd4412102cfc43c675370c3f0050ffc1d340493fc74e6ca8bb6280c27fd336372f41b2b20ffffffff047803000000000000535101400100015101b101b26114ebaa76479d676a18e9967b09d4939bc90c4b3246005179517a7561587905626164676587695979a9517987695a795a79ac77777777777777777777776a0820a10700000000007803000000000000535101400100015101b101b2611474b0bbf85657c5c3e412fcf003ee713ade38c2a4005179517a7561587905626164676587695979a9517987695a795a79ac77777777777777777777776a0820aa4400000000007803000000000000535101400100015101b101b2611452dee27ceab2761f3e72176b2dfad30f0f03bacd005179517a7561587905626164676587695979a9517987695a795a79ac77777777777777777777776a083cc43a000000000069713600000000001976a91452dee27ceab2761f3e72176b2dfad30f0f03bacd88ac00000000"
}' http://127.0.0.1:7789/v1/touchstone/sendrawtransaction
```

- Response

  - In vout, `pretxid` and `preindex` will always be an empty str and -1

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

- Description

Inspect how many badge tokens were included in a transaction.

- Params

| param | required | note           |
| ----- | -------- | -------------- |
| txid  | true     | transaction id |

- Request

```shell
curl -X POST --data '{
    "txid":"443110747ccd782c7dc480daec7faee63533c4c2cdf7231966d7a0cc61036132"
}' http://127.0.0.1:7789/v1/touchstone/gettxinventory
```

- Response
  - In vout, `pretxid` and `preindex` will always be an empty string and -1

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

- Description

Get the badge related utxos for an address.

- Params

| param      | required | note              |
| ---------- | -------- | ----------------- |
| addr       | true     | address           |
| badge_code | true     | badge code        |
| offset     | false    | offset, default 0 |
| limit      | false    | limit, default 10 |

- Request

```shell
curl -X POST --data '{
    "badge_code":"e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
    "addr":"1LRKoKfHef3DMZ7aLqAiwsf1a3TQYQ4G9i"
}' http://127.0.0.1:7789/v1/touchstone/getaddrutxos
```

- Response
  - In vout, `pretxid` and `preindex` will always be an empty string and -1

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

- Description

Get the total balance of a specific badge token for an address.

- Params

| param      | required | note       |
| ---------- | -------- | ---------- |
| addr       | true     | address    |
| badge_code | true     | badge code |

- Request

```shell
curl -X POST --data '{
    "badge_code":"e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
    "addr":"1LRKoKfHef3DMZ7aLqAiwsf1a3TQYQ4G9i"
}' http://127.0.0.1:7789/v1/touchstone/getaddrbalance
```

- Response

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

- Description

Get the badge transaction history for a certain badge token for an address.
Negative values are outgoing transactions.

- Params

| param      | required | note              |
| ---------- | -------- | ----------------- |
| addr       | true     | address           |
| badge_code | true     | badge code        |
| offset     | false    | offset, default 0 |
| limit      | false    | limit, default 10 |

- Request

```shell
curl -X POST --data '{
    "badge_code":"e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
    "addr":"1PLuQQPRBcpDurPc9bZAw5pcePgCNatCfG"
}' http://127.0.0.1:7789/v1/touchstone/getaddrinventorys
```

- Response

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

- Description

Associate an address with some user and wallet information. This information will be stored by Touchstone nodes. Query this info with [getuserbalance](#getuserbalance).

- Params

| param      | required | note                                                        |
| ---------- | -------- | ----------------------------------------------------------- |
| userid     | true     | user id                                                     |
| appid      | true     | Recommended to indicate the tpye or use case of the address |
| user_index | true     | Useful if a user has more than one wallet                   |
| addr       | true     | address                                                     |

- Request

```shell
curl -X POST --data '{
    "userid":1,
    "appid":"auto pay",
    "user_index":0,
    "addr":"1PLuQQPRBcpDurPc9bZAw5pcePgCNatCfG"
}' http://127.0.0.1:7789/v1/touchstone/setaddrinfo
```

- Response

```json
{
	"code": 0,
	"msg": "",
	"data": null
}
```

### <span id="getuserutxos">getuserutxos</span>

- Description

Get badge related utxos for a specific badge and user.

- Params

| param      | required | note                          |
| ---------- | -------- | ----------------------------- |
| appid      | true     | app id set by setaddrinfo     |
| userid     | true     | user id set by setaddrinfo    |
| user_index | true     | user index set by setaddrinfo |
| badge_code | true     | badge code                    |
| offset     | false    | offset, default 0             |
| limit      | false    | limit, default 10             |

- Request

```shell
curl -X POST --data '{
    "userid":1,
    "appid":"auto pay",
    "user_index":1,
    "badge_code":"e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700"
}' http://127.0.0.1:7789/v1/touchstone/getuserutxos
```

- Response

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

- Description

Query user info and balance of a certain badge.

- Params

| param      | required | note                          |
| ---------- | -------- | ----------------------------- |
| appid      | true     | app id set by setaddrinfo     |
| userid     | true     | user id set by setaddrinfo    |
| user_index | true     | user index set by setaddrinfo |
| badge_code | true     | badge code                    |

- Request

```shell
curl -X POST --data '{
    "userid":1,
    "appid":"auto pay",
    "user_index":1,
    "badge_code":"e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700"
}' http://127.0.0.1:7789/v1/touchstone/getuserbalance
```

- Response

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

- Description

Get the transaction history for a certain user and certain badge.

- Params

| param      | required | note                          |
| ---------- | -------- | ----------------------------- |
| appid      | true     | app id set by setaddrinfo     |
| userid     | true     | user id set by setaddrinfo    |
| user_index | true     | user index set by setaddrinfo |
| badge_code | true     | badge code                    |
| offset     | false    | offset, default 0             |
| limit      | false    | limit, default 10             |

- Request

```shell
curl -X POST --data '{
    "userid":1,
    "appid":"auto pay",
    "user_index":1,
    "badge_code":"e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700"
}' http://127.0.0.1:7789/v1/touchstone/getuserinventorys
```

- Response

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

- Description

Create an unsigned insufficient fee transaction

- Params

| param        | required | note                                                                |
| ------------ | -------- | ------------------------------------------------------------------- |
| appid        | true     | app id set by setaddrinfo                                           |
| userid       | true     | user id set by setaddrinfo                                          |
| user_index   | true     | user index set by setaddrinfo                                       |
| badge_code   | true     | badge code                                                          |
| addr_amounts | false    | receiver's address and amount. Leave empty for burn or collect utxo |
| amount2burn  | false    | amount to burn                                                      |
| change_addr  | true     | address to send change to                                           |

- Request

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

- Response

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
