/*
Copyright 2021 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
// 需要家
// Org2のユーザで実行

package main

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"time"
	"net/http"
	"encoding/json"
	"bytes"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	mspID         = "Org1MSP"
	cryptoPath    = "../../../test-network/organizations/peerOrganizations/org1.example.com"
	certPath      = cryptoPath + "/users/User1@org1.example.com/msp/signcerts/cert.pem"
	keyPath       = cryptoPath + "/users/User1@org1.example.com/msp/keystore/"
	tlsCertPath   = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint  = "localhost:7051"
	gatewayPeer   = "peer0.org1.example.com"
	channelName   = "mychannel"
	chaincodeName = "basic"
)

// Rest APIからの入力
type Input struct {
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	User            string    `json:"user"`
	Category string `json:"category"`
}

// 現在時刻の取得
var now = time.Now()

func main() {
	// ポート番号8080のcreateTokenで受け付け
	log.Println("============ application-golang starts ============")
	http.HandleFunc("/createToken", handler)
	http.ListenAndServe(":8080", nil)
	log.Println("============ application-golang ends ============")
}

func handler(w http.ResponseWriter, r *http.Request) {

	// POSTメソッドのみ受け付け
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed) //405
		w.Write([]byte("Only POST"))
		return
	}

	// application/json; charset=utf-8のみ受け付け
	if r.Header.Get("Content-Type") != "application/json; charset=utf-8" {
		w.WriteHeader(http.StatusBadRequest) //400
		w.Write([]byte("Only json"))
		return
	}

	// HTTPリクエストのBodyを処理
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) //400
		w.Write([]byte(err.Error()))
		return
	}

	// HTTPリクエストのBodyを入れる構造体
	var requestInput Input

	// HTTPリクエストのBodyを構造体に変換
	err = json.Unmarshal(body, &requestInput)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError) //500
		w.Write([]byte(err.Error()))
		return
	}

	// トークン作成
	createEnergy, timestamp, err := createContract(requestInput)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	// 作成したトークンをHTTPレスポンスにする
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err = enc.Encode(&createEnergy); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte(buf.String()))

	// エラーが無かった場合の処理
	if createEnergy.Error == "" {
		// マップに作成したトークンのデータを渡す
		go HttpPostCreatedToken(createEnergy)
		// オークション結果確認
		go auctionContract(createEnergy, timestamp, requestInput)
	}

}

// コネクションを確立しトークンを作成
func createContract(input Input) (Energy, time.Time, error) {
	var energy Energy
	var timestamp time.Time

	// Grpcを利用
	clientConnection := newGrpcConnection()
	defer clientConnection.Close()

	id := newIdentity()
	sign := newSign()

	// ゲートウェイ接続確立
	gateway, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConnection),
		// タイムアウトの設定
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		return energy, timestamp, err
	}
	defer gateway.Close()

	network := gateway.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	// 作成したトークン
	energy, timestamp = Create(contract, input)
	
	return energy, timestamp, nil

}

// コネクションを確立しオークション結果を確認
func auctionContract(energy Energy, timestamp time.Time, input Input) {
	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	clientConnection := newGrpcConnection()
	defer clientConnection.Close()

	id := newIdentity()
	sign := newSign()

	// Create a Gateway connection for a specific client identity
	gateway, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gateway.Close()

	network := gateway.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	// オークション結果の確認
	Auction(contract, energy, timestamp, input)
}

// newGrpcConnection creates a gRPC connection to the Gateway server.
func newGrpcConnection() *grpc.ClientConn {
	certificate, err := loadCertificate(tlsCertPath)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	connection, err := grpc.Dial(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}

	return connection
}

// newIdentity creates a client identity for this Gateway connection using an X.509 certificate.
func newIdentity() *identity.X509Identity {
	certificate, err := loadCertificate(certPath)
	if err != nil {
		panic(err)
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		panic(err)
	}

	return id
}

func loadCertificate(filename string) (*x509.Certificate, error) {
	certificatePEM, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}
	return identity.CertificateFromPEM(certificatePEM)
}

// newSign creates a function that generates a digital signature from a message digest using a private key.
func newSign() identity.Sign {
	files, err := ioutil.ReadDir(keyPath)
	if err != nil {
		panic(fmt.Errorf("failed to read private key directory: %w", err))
	}
	privateKeyPEM, err := ioutil.ReadFile(path.Join(keyPath, files[0].Name()))

	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		panic(err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		panic(err)
	}

	return sign
}
