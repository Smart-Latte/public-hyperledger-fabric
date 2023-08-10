/*
Copyright 2021 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

/*
需要家のアプリケーション
Org2のユーザで実行
*/

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
	mspID         = "Org2MSP"
	cryptoPath    = "../../../test-network/organizations/peerOrganizations/org2.example.com"
	certPath      = cryptoPath + "/users/User1@org2.example.com/msp/signcerts/cert.pem"
	keyPath       = cryptoPath + "/users/User1@org2.example.com/msp/keystore/"
	tlsCertPath   = cryptoPath + "/peers/peer0.org2.example.com/tls/ca.crt"
	peerEndpoint  = "localhost:9051"
	gatewayPeer   = "peer0.org2.example.com"
	channelName   = "mychannel"
	chaincodeName = "basic"
)

// Rest APIからの入力
type Input struct {
	Token int `json:"Token"`
	BatteryLife int `json:"batteryLife"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	User            string    `json:"user"`
}

// Rest APIの戻り値
type Return struct {
	DocType          string    `json:"DocType`
	UnitPrice        float64   `json:"Unit Price"`
	BidPrice         float64   `json:"Bid Price"`
	GeneratedTime    time.Time `json:"Generated Time"`
	AuctionStartTime time.Time `json:"Auction Start Time"`
	BidTime          time.Time `json:"Bid Time"`
	ID               string    `json:"ID"`
	LargeCategory    string    `json:"LargeCategory"`
	Latitude         float64   `json:"Latitude"`
	Longitude        float64   `json:"Longitude"`
	Owner            string    `json:"Owner"`
	Producer         string    `json:"Producer"`
	SmallCategory    string    `json:"SmallCategory"`
	Status           string    `json:"Status"`
	Error string `json:"Error"`
}

// 現在時刻の取得
var now = time.Now()

func main() {

	// ポート番号9080のbidOnTokenで受け付け
	log.Println("============ application-golang starts ============")
	http.HandleFunc("/bidOnToken", handler)
	http.ListenAndServe(":9080", nil)
	log.Println("============ application-golang ends ============")
}

// HTTPリクエストを処理
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
		fmt.Println("Unmarshal error")
		fmt.Println(err.Error())
		w.Write([]byte(err.Error()))
		return
	}

	// 入札が成功したトークンのリストを取得
	successList, err := bidContract(requestInput)
	if err != nil {
		fmt.Println("bidContract")
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	// トークンのリストをHTTPレスポンスに入れるため変換
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err = enc.Encode(&successList); err != nil {
		fmt.Println("Encode")
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte(buf.String()))

	// 入札が成功したトークンがあるなら、入札結果を確認
	if len(successList) > 0 {
		go bidResultContract(successList, requestInput)
	}
}

// コネクションを確立し、入力データに合わせ、入札を実行
func bidContract(input Input) ([]Energy, error) {
	var energies []Energy

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
		fmt.Println("gatewayerror")
		return energies, err
	}
	defer gateway.Close()

	network := gateway.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	// 入札が成功したトークンのリストが返る
	successList, err := Buy(contract, input)
	if (err != nil) {
		fmt.Println("buy error")
		return energies, err
	}
	return successList, nil
}

// コネクションを確立し入札結果を確認
func bidResultContract(successList []Energy, input Input) {
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
		fmt.Println(err)
	}
	defer gateway.Close()

	network := gateway.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	// 入札結果を確認する
	BidResult(contract, successList, input)
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
