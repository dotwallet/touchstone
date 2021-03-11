package util

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
	TX_VERSION                 = 2
	BADGE_FLAG                 = "badge"
	BADGE_CODE_PART_HEX_PREFIX = "5101400100015101b101b26114"
	BADGE_CODE_PART_HEX_SUFFIX = "005179517a7561587905626164676587695979a9517987695a795a79ac7777777777777777777777"
	PUBKEY_HASH_LEN            = 20
	HEX_PUBKEY_HASH_LEN        = 2 * PUBKEY_HASH_LEN
	BADGE_CODE_PART_LEN        = len(BADGE_CODE_PART_HEX_PREFIX)/2 + len(BADGE_CODE_PART_HEX_SUFFIX)/2 + PUBKEY_HASH_LEN
	BADGE_DATA_PART_HEX_PRIFIX = "6a08"
	BADGE_DATA_LEN             = 8
	BADGE_DATA_PART_LEN        = len(BADGE_DATA_PART_HEX_PRIFIX)/2 + BADGE_DATA_LEN // op_return + op_8 + BADGE_DATA_LEN
	BADGE_LOCKING_SCRIPT_LEN   = BADGE_CODE_PART_LEN + BADGE_DATA_PART_LEN

	MAX_CREATE_BADGE_VALUE = 9223372036854775807
)

func GetHashByte(hash chainhash.Hash) []byte {
	hexHash := hash.String()
	bytes, err := hex.DecodeString(hexHash)
	if err != nil {
		panic(err)
	}
	return bytes
}

func SeserializeMsgTxStr(msgtx *wire.MsgTx) string {
	return hex.EncodeToString(SeserializeMsgTxBytes(msgtx))
}
func SeserializeMsgTxBytes(msgtx *wire.MsgTx) []byte {
	buf := make([]byte, 0, msgtx.SerializeSize())
	buff := bytes.NewBuffer(buf)
	msgtx.Serialize(buff)
	return buff.Bytes()
}

func DeserializeTxStr(rawtx string) (*wire.MsgTx, error) {
	txbytes, err := hex.DecodeString(rawtx)
	if err != nil {
		return nil, err
	}
	return DeserializeTxBytes(txbytes)

}

func DeserializeTxBytes(txbytes []byte) (*wire.MsgTx, error) {
	msgtx := wire.NewMsgTx(TX_VERSION)
	err := msgtx.Deserialize(bytes.NewReader(txbytes))
	if err != nil {
		return nil, err
	}
	return msgtx, nil
}

func Int64ToBytes(num int64) ([]byte, error) {
	s1 := make([]byte, 0, 8)
	buf := bytes.NewBuffer(s1)
	err := binary.Write(buf, binary.LittleEndian, num)
	if err != nil {
		return nil, err
	}
	result := buf.Bytes()
	return result, nil
}

func CreateBadgeLockScript(address btcutil.Address, value int64) ([]byte, error) {
	if value < 0 {
		return nil, errors.New("error value")
	}
	prefix, err := hex.DecodeString(BADGE_CODE_PART_HEX_PREFIX)
	if err != nil {
		panic(err)
	}
	suffix, err := hex.DecodeString(BADGE_CODE_PART_HEX_SUFFIX)
	if err != nil {
		panic(err)
	}
	result := prefix
	result = append(result, address.ScriptAddress()...)
	result = append(result, suffix...)

	valueByte, err := Int64ToBytes(value)
	if err != nil {
		return nil, err
	}

	dataPart, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddData(valueByte).Script()
	if err != nil {
		panic(err)
	}
	result = append(result, dataPart...)
	return result, nil
}

type BadgeVout struct {
	BadgeValue int64
	Address    btcutil.Address
}

//todo
func ParseBadgeVoutScript(script []byte, net *chaincfg.Params) (*BadgeVout, error) {
	if len(script) != BADGE_LOCKING_SCRIPT_LEN {
		return nil, errors.New("not badge vout 1")
	}

	hexScript := hex.EncodeToString(script)
	if !strings.HasPrefix(hexScript, BADGE_CODE_PART_HEX_PREFIX) {
		return nil, errors.New("not badge vout 2")
	}

	index := strings.Index(hexScript, BADGE_CODE_PART_HEX_SUFFIX)
	if index != len(BADGE_CODE_PART_HEX_PREFIX)+HEX_PUBKEY_HASH_LEN {
		return nil, errors.New("not badge vout 3")
	}
	if hexScript[146] != 0x36 ||
		hexScript[147] != 0x61 ||
		hexScript[148] != 0x30 ||
		hexScript[149] != 0x38 {
		return nil, errors.New("not badge vout 4")
	}
	address, err := btcutil.NewAddressPubKeyHash(script[len(BADGE_CODE_PART_HEX_PREFIX)/2:len(BADGE_CODE_PART_HEX_PREFIX)/2+PUBKEY_HASH_LEN], net)
	if err != nil {
		return nil, err
	}

	value := int64(binary.LittleEndian.Uint64(script[(len(BADGE_CODE_PART_HEX_PREFIX)+HEX_PUBKEY_HASH_LEN+len(BADGE_CODE_PART_HEX_SUFFIX)+len(BADGE_DATA_PART_HEX_PRIFIX))/2:]))
	if value < 0 {
		return nil, errors.New("error value")
	}
	return &BadgeVout{
		BadgeValue: value,
		Address:    address,
	}, nil
}
