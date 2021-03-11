# touchstone

&emsp;&emsp;touchstone is a tool to verify badge transaction.

## <span id="index">index</span>
- [index](#index)

- [introduce](#introduce)

- [how to build](#howtobuild)

- [how to run](#howtorun)

- [api method](#apimethod)
## <span id="introduce">introduce</span>

- [background](#background)
- [badge script](#badgescript)
- [badge protocol](#badgeprotocol)
### <span id="background">background</span>
&emsp;&emsp;for now,in the current bsv community, it has not been able to propose a practical complete layer one token issuance solution.even more and more partial layer one solutions try their best to do as many as possible jobs in layer one,it still can't just let user simply trust a transaction come from net without any layer two verification.and most partial layer one solutions also cause more fee,wait longer,worse user experience. some of them can't even transfer decentralized. so if we can't get rid of layer two verification and layer two verification cost basically same no matter what solution you take,why not just let layer two verification take more responsibility? it uses the simpler way to implement same function with no additional consumption and bring the same or even better user experience.

&emsp;&emsp;and so ,badge came into being.

&emsp;&emsp;badge is a token issuance solution,which mainly in layer two. it is base on utxo model,and allow anyone transfer decentralized,verify decentralized. and touchstone is a tool to verify badge transaction.

### <span id="badgescript">badge script</span>

- badge vout

&emsp;&emsp;badge script was generated via scrypt

```js
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

&emsp;&emsp;here is a badge vout example:
```
1 64 0 81 -49 -50 OP_NOP <pubkey hash> 0 1 OP_PICK 1 OP_ROLL OP_DROP OP_NOP 8 OP_PICK 6261646765 OP_EQUAL OP_VERIFY 9 OP_PICK OP_HASH160 1 OP_PICK OP_EQUAL OP_VERIFY 10 OP_PICK 10 OP_PICK OP_CHECKSIG OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_NIP OP_RETURN 1027000000000000
```
it represent `<pubkey hash>`  own `10000`(little end 1027000000000000) of some kind badge 

- badge vin 

&emsp;&emsp;for vout above,you will need vin like:
```
<sig> <pubkey> <badge flag>
```
to unlock. `<badge flag>` is "badge" .



### <span id="badgeprotocol">badge protocol</span>

&emsp;&emsp;consider we have a badge utxo set,let see what will happen when transtions come. for now ,the utxo set is empty.

- issuance

&emsp;&emsp;if there is a `tx1` like this:

| vin           | vout                     |
| ------------- | ------------------------ |
| not badge vin | badge vout 1, contain 50 |
| ...           | badge vout 2, contain 40 |
|               | badge vout 3, contain 30 |
|               | badge vout 4, contain 30 |
|               | ...                      |

with at least one badge format vout and no badge format vin,we consider it is a badge issuance. vout order does not matter. it can also contain other kind srcipt.

and now ,our utxo set got 

`<badge vout 1,contain 50>` `<badge vout 2,contain 40>` `<badge vout 3,contain 30>` `<badge vout 4,contain 30>` , the transaction issue 150 badge.

and another `tx2` like:

| vin                  | vout                              |
| -------------------- | --------------------------------- |
| not badge format vin | badge format vout 5, contain 1000 |
| ...                  | ...                               |

our utxo set will be:  

`<badge vout 1,contain 50>` `<badge vout 2,contain 40>` `<badge vout 3,contain 30>` `<badge vout 4,contain 30>` `<badge vout 5,contain 1000>`,

but we should know `<badge format vout 5,contain 1000>>` is different from others. may be we can call them 

`<badge vout 1,contain 50 of b1>` `<badge vout 2,contain 40 of b1>` `<badge vout 3,contain 30 of b1>` `<badge vout 4,contain 30 of b1>` `<badge vout 5,contain 1000 of b2>`


&emsp;&emsp;

- transfer 

&emsp;&emsp;if alice wants to transfer bob 70 b1,carol 10 b1,and get change, we may see `tx3` like:

| vin                             | vout                            |
| ------------------------------- | ------------------------------- |
| <badge vout 1,contain 50 of b1> | <badge vout 6,contain 70 of b1> |
| <badge vout 2,contain 40 of b1> | <badge vout 7,contain 10 of b1> |
| ...                             | <badge vout 8,contain 10 of b1> |


our utxo set will be:  

`<badge vout 3,contain 30 of b1>` `<badge vout 4,contain 30 of b1>` `<badge vout 5,contain 1000 of b2>` `<badge vout 6,contain 70 of b1>` `<badge vout 7,contain 10 of b1>` `<badge vout 8,contain 10 of b1>`

&emsp;&emsp;if we see `tx4`

| vin                             | vout                            |
| ------------------------------- | ------------------------------- |
| <badge vout 3,contain 30 of b1> | <badge vout 9,contain 41 of b1> |
| <badge vout 7,contain 10 of b1> | ...                             |
| ...                             |                                 |

because vout badge count is greater than vin,so  `<badge vout 9,contain 41 of b1>` will not be token,but `<badge vout 3,contain 30 of b1>` `<badge vout 7,contain 20 of b1>` is still gone.


our utxo set will be:  

`<badge vout 4,contain 30 of b1>` `<badge vout 5,contain 1000 of b2>` `<badge vout 6,contain 70 of b1>` `<badge vout 8,contain 10 of b1>`


&emsp;&emsp;if we see tx5

| vin                               | vout                              |
| --------------------------------- | --------------------------------- |
| <badge vout 4,contain 30 of b1>   | <badge vout 10,contain 1030 of ?> |
| <badge vout 5,contain 1000 of b2> | ...                               |
| ...                               |                                   |


even vout badge count is not greater than vin,but vin contains `b1` and `b2`,which is not support for now. we may upgrade protocol to support this case in the future, but now ,`<badge vout 10,contain 1030 of ?>` will not be token and `<badge vout 4,contain 30 of b1>` `<badge vout 5,contain 1000 of b2>` is gone.

our utxo set will be:  

`<badge vout 6,contain 70 of b1>` `<badge vout 8,contain 10 of b1>`


- burn

&emsp;&emsp;if we see tx6

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

if there is a transaction,vout badge count is less than vin,we can easily find that total badge utxo set contain is getting less. so we can use `tx6` and `tx7` to burn the badge. by the way, `tx4` and `tx5` also reduce the count of badge, you may use it too.



## <span id="howtobuild">how to build</span>

-  make sure you got golang and all grpc protobuf support in your env
```shell
sh build.sh
```
## <span id="howtorun">how to run</span>


to run a touchstone node , you at least need 
- mongo
- mapi support
- a private key of bitcoin for yourself
- known peers pubkey and host
- let peers have your pubkey

here is a example for config
```json
{
    "Env": "mainnet",
    "MongoHost": "127.0.0.1:27017",
    "MempoolHost": "api.ddpurse.com",
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
- this version of code only support mapi provided by mempool, you can easily replace it by any provider. just implement `MapiClientAdaptor` in `mapi/mapi_client.go`,and modify code in `main.go`
```golang
	mapiClient, err := mapi.NewMempoolMapiClient(config.MempoolHost, config.MempoolPkiMnemonic, config.MempoolPkiMnemonicPassword)
```

for now ,touchstone relay on mapi to verify transaction.In the future,we may access bitcoin p2p network

## <span id="apimethod">api method</span>

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
