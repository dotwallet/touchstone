package conf

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
)

const (
	ENV_RGTEST  = "regtest"
	ENV_MAINNET = "mainnet"

	REGTEST_START_HEIGHT = 0
	MAINNET_START_HEIGHT = 650000

	PARTITION_BLOCK_COUNT = 10

	COMPARE_PARTITIONS_COUNT = 10

	RE_CONPUTE_PARTITION_COUNT = 1
)

type PeerConfig struct {
	Host   string
	Pubkey string
}

type Config struct {
	Env                        string
	MongoHost                  string
	MempoolHost                string
	MempoolPkiMnemonic         string
	MempoolPkiMnemonicPassword string
	ServerPrivatekey           string
	PeersConfigs               []*PeerConfig
	P2pHost                    string
	HttpHost                   string
	DbName                     string
}

var GStartHeight *int64

var GNetParam *chaincfg.Params

func InitGConfig(env string) error {
	startHeight := int64(0)
	switch env {
	case ENV_RGTEST:
		startHeight = REGTEST_START_HEIGHT
		GNetParam = &chaincfg.RegressionNetParams
	case ENV_MAINNET:
		startHeight = MAINNET_START_HEIGHT
		GNetParam = &chaincfg.MainNetParams
	default:
		errStr := fmt.Sprintf("not support env %s", env)
		return errors.New(errStr)
	}
	GStartHeight = &startHeight
	return nil
}
