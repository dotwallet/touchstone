package interceptor

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/dotwallet/touchstone/conf"
	"github.com/dotwallet/touchstone/message"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

const (
	KEY_PUBKEY     = "pubkey"
	KEY_TIMESTAMP  = "timestamp"
	KEY_SIGNATURE  = "signature"
	AUTH_HEAD_LONG = 4
)

func AllowPubkeyFromConfigs() {

}

func VerifyTimestampSig(pubkeyByte []byte, sigBytes []byte, timestamp int64) error {
	now := time.Now().Unix()
	if now-timestamp > 10 {
		return errors.New("sig expired")
	}
	if now < timestamp {
		return errors.New("error timestamp")
	}
	byteBuf := bytes.NewBuffer(make([]byte, 0, 8))
	err := binary.Write(byteBuf, binary.LittleEndian, timestamp)
	if err != nil {
		return err
	}

	pubkey, err := btcec.ParsePubKey(pubkeyByte, btcec.S256())
	if err != nil {
		return err
	}
	sig, err := btcec.ParseSignature(sigBytes, btcec.S256())
	if err != nil {
		return err
	}
	ok := sig.Verify(byteBuf.Bytes(), pubkey)
	if !ok {
		return errors.New("verify fail")
	}
	return nil
}

type AuthPerRPCCredential struct {
	privateKey *btcec.PrivateKey
}

func NewAuthPerRPCCredential(privateKey *btcec.PrivateKey) *AuthPerRPCCredential {
	return &AuthPerRPCCredential{
		privateKey: privateKey,
	}
}

func (this AuthPerRPCCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	timestamp := time.Now().Unix()
	byteBuf := bytes.NewBuffer(make([]byte, 0, 8))
	err := binary.Write(byteBuf, binary.LittleEndian, timestamp)
	if err != nil {
		return nil, err
	}
	numByte := byteBuf.Bytes()
	sig, err := this.privateKey.Sign(numByte)
	if err != nil {
		return nil, err
	}
	timestampStr := strconv.FormatInt(timestamp, 10)
	pubkeyHex := hex.EncodeToString(this.privateKey.PubKey().SerializeCompressed())
	return map[string]string{
		KEY_PUBKEY:    pubkeyHex,
		KEY_TIMESTAMP: timestampStr,
		KEY_SIGNATURE: hex.EncodeToString(sig.Serialize()),
	}, nil
}

func (this AuthPerRPCCredential) RequireTransportSecurity() bool {
	return false
}

type AuthInterceptor struct {
	allowPubkeys map[string]bool
}

func NewAuthInterceptor(PeerConfigs []*conf.PeerConfig) *AuthInterceptor {
	authInterceptor := &AuthInterceptor{
		allowPubkeys: make(map[string]bool),
	}
	for _, PeerConfig := range PeerConfigs {
		authInterceptor.allowPubkeys[PeerConfig.Pubkey] = true
	}
	return authInterceptor
}

func (this *AuthInterceptor) Intercept(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("not header")
	}
	pubkeyHex, ok := md[KEY_PUBKEY]
	if !ok {
		return nil, errors.New("no pubkey")
	}
	_, ok = this.allowPubkeys[pubkeyHex[0]]
	if !ok {
		return nil, errors.New("not support pubkey")
	}
	pubkeyByte, err := hex.DecodeString(pubkeyHex[0])
	if err != nil {
		return nil, err
	}
	sigHex, ok := md[KEY_SIGNATURE]
	if !ok {
		return nil, errors.New("not sig")
	}
	sigBytes, err := hex.DecodeString(sigHex[0])
	if err != nil {
		return nil, err
	}
	timestampStr, ok := md[KEY_TIMESTAMP]
	if !ok {
		return nil, errors.New("no timestamp")
	}
	timestamp, err := strconv.ParseInt(timestampStr[0], 10, 64)
	if err != nil {
		return nil, err
	}
	err = VerifyTimestampSig(pubkeyByte, sigBytes, timestamp)
	if err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

type AuthInfo struct {
	Info string
}

func (this *AuthInfo) AuthType() string {
	return this.Info
}

type AuthCredential struct {
	privateKey   *btcec.PrivateKey
	allowPubkeys map[string]bool
}

func NewClientAuthCredential(privateKey *btcec.PrivateKey) *AuthCredential {
	return &AuthCredential{
		privateKey: privateKey,
	}
}

func NewServerAuthCredential(PeerConfigs []*conf.PeerConfig) *AuthCredential {
	allowPubkeys := make(map[string]bool)
	for _, PeerConfig := range PeerConfigs {
		allowPubkeys[PeerConfig.Pubkey] = true
	}
	return &AuthCredential{
		allowPubkeys: allowPubkeys,
	}
}

func (this *AuthCredential) ClientHandshake(c context.Context, s string, conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	timestamp := time.Now().Unix()
	byteBuf := bytes.NewBuffer(make([]byte, 0, 8))
	err := binary.Write(byteBuf, binary.LittleEndian, timestamp)
	if err != nil {
		conn.Close()
		return conn, nil, err
	}
	numByte := byteBuf.Bytes()
	sig, err := this.privateKey.Sign(numByte)
	if err != nil {
		conn.Close()
		return conn, nil, err
	}
	authRequest := &message.AuthRequest{
		Signature: sig.Serialize(),
		Pubkey:    this.privateKey.PubKey().SerializeCompressed(),
		Timestamp: timestamp,
	}
	payload, err := proto.Marshal(authRequest)
	if err != nil {
		conn.Close()
		return conn, nil, err
	}
	authReaBuf := bytes.NewBuffer(make([]byte, 0, 4))
	err = binary.Write(authReaBuf, binary.LittleEndian, uint32(len(payload)))
	if err != nil {
		conn.Close()
		return conn, nil, err
	}
	n, err := authReaBuf.Write(payload)
	if err != nil {
		conn.Close()
		return conn, nil, err
	}
	if n != len(payload) {
		conn.Close()
		return conn, nil, errors.New("n != len(payload)")
	}
	authReqByte := authReaBuf.Bytes()
	n, err = conn.Write(authReqByte)
	if err != nil {
		conn.Close()
		return conn, nil, err
	}
	if n != len(authReqByte) {
		conn.Close()
		return conn, nil, errors.New("n != len(authReqByte)")
	}
	return conn, &AuthInfo{
		Info: "ClientHandshakeInfo",
	}, nil
}

type ConnReader struct {
	conn     net.Conn
	buf      []byte
	writePos int
	readPos  int
}

func NewConnReader(conn net.Conn) *ConnReader {
	return &ConnReader{
		buf:      make([]byte, 1024),
		conn:     conn,
		writePos: 0,
		readPos:  0,
	}
}

func (this *ConnReader) CheckAndMakeRoom(needSize int) {
	if needSize <= len(this.buf[this.writePos:]) {
		return
	}
	if needSize <= len(this.buf[this.writePos:])+len(this.buf[:this.readPos]) {
		copy(this.buf[0:this.writePos-this.readPos], this.buf[this.readPos:this.writePos])
		this.writePos -= this.readPos
		this.readPos = 0
		return
	}
	cap := len(this.buf) * 2
	if cap < needSize {
		cap = needSize * 2
	}
	newBuf := make([]byte, cap)
	copy(newBuf, this.buf[this.readPos:this.writePos])
	this.writePos -= this.readPos
	this.readPos = 0
	return

}

func (this *ConnReader) Read(buf []byte) error {
	if len(buf) <= len(this.buf[this.readPos:this.writePos]) {
		// fmt.Println("Read 1")
		copy(buf, this.buf[this.readPos:this.writePos])
		this.readPos += len(buf)
		return nil
	}
	this.CheckAndMakeRoom(len(buf))
	for {
		// fmt.Println("Read 2 ", this.buf)
		n, err := this.conn.Read(this.buf[this.writePos:])
		this.writePos += n
		if err != nil {
			return err
		}
		if len(buf) <= len(this.buf[this.readPos:this.writePos]) {
			copy(buf, this.buf[this.readPos:this.writePos])
			this.readPos += len(buf)
			return nil
		}
	}

}

func (this *AuthCredential) ServerHandshake(conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	err := conn.SetReadDeadline(time.Now().Add(time.Second * 10))
	if err != nil {
		conn.Close()
		return conn, nil, err
	}

	reader := NewConnReader(conn)
	head := make([]byte, 4)
	err = reader.Read(head)
	if err != nil {
		conn.Close()
		return conn, nil, err
	}
	bodyLen := binary.LittleEndian.Uint32(head)
	if bodyLen > 1000 {
		if err != nil {
			conn.Close()
			return conn, nil, errors.New("not support body size")
		}
	}
	body := make([]byte, bodyLen)
	err = reader.Read(body)
	if err != nil {
		conn.Close()
		return conn, nil, err
	}
	request := &message.AuthRequest{}
	err = proto.Unmarshal(body, request)
	if err != nil {
		conn.Close()
		return conn, nil, err
	}
	err = VerifyTimestampSig(request.Pubkey, request.Signature, request.Timestamp)
	if err != nil {
		conn.Close()
		return conn, nil, err
	}
	err = conn.SetReadDeadline(time.Time{})
	if err != nil {
		conn.Close()
		return conn, nil, err
	}
	return conn, &AuthInfo{
		Info: "ServerHandshakeInfo",
	}, nil
}

func (this *AuthCredential) Clone() credentials.TransportCredentials {
	return &AuthCredential{
		privateKey:   this.privateKey,
		allowPubkeys: this.allowPubkeys,
	}
}

func (this *AuthCredential) Info() credentials.ProtocolInfo {
	return credentials.ProtocolInfo{}
}

func (this *AuthCredential) OverrideServerName(string) error {
	return nil
}
