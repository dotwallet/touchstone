# touchstone
## how to build
-  make sure you got golang and all grpc protobuf support in your env
```shell
sh build.sh
```
## how to run a touchstone node

### to run a touchstone node , you at least need 
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

## http method

### sendrawtransaction
- req
```shell
curl -X POST --data '{
    "rawtx":"0200000001a66cb4e7620473d559eee1ef7f9328784444b9506798e4f57a7ca6340df61c5a330000006b483045022100d87a198987fdefd038eb016bf791ad7ed4882e26b30471bb6ee0ce5f66f920ae02205104444d800776a408c50236b038b2a7e3fe9cf2a701b3f02f70b95379b5fda041210253ea79eb987d96e7c9f91a7e8a212871baaf59ad022692af9fa33db7e62f32e7ffffffff027803000000000000535101400100015101b101b26114f5166a2da65a519a1620242ac793e92dcfc62652005179517a7561587905626164676587695979a9517987695a795a79ac77777777777777777777776a080010a5d4e8000000a2bd8b00000000001976a91447219696fc1b4fc2cd07a09f618cb0e6a205775988ac00000000"
}' http://127.0.0.1:7789/v1/touchstone/sendrawtransaction
```
- rsp 
```json
{
    "code": 0,
    "msg": "",
    "data": {
        "vins": [],
        "vouts": [
            {
                "addr": "1PLuQQPRBcpDurPc9bZAw5pcePgCNatCfG",
                "index": 0,
                "value": 1000000000000,
                "badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
                "timestamp": 0
            }
        ]
    }
}
```
### gettxinventory
- req
```shell
curl -X POST --data '{
    "txid":"e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700"
}' http://127.0.0.1:7789/v1/touchstone/gettxinventory
```
- rsp 
```json
{
    "code": 0,
    "msg": "",
    "data": {
        "vins": [],
        "vouts": [
            {
                "addr": "1PLuQQPRBcpDurPc9bZAw5pcePgCNatCfG",
                "index": 0,
                "value": 1000000000000,
                "badge_code": "e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700",
                "timestamp": 0
            }
        ]
    }
}
```