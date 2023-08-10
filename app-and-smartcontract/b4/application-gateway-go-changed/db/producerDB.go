package main

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path"
	"time"
	//"sync"
	"log"
	"encoding/json"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"database/sql"
	//"os"

	_ "github.com/mattn/go-sqlite3"
)

const (
	/*
	mspID         = "Org1MSP"
	cryptoPath    = "../../../../test-network/organizations/peerOrganizations/org1.example.com"
	certPath      = cryptoPath + "/users/User1@org1.example.com/msp/signcerts/cert.pem"
	keyPath       = cryptoPath + "/users/User1@org1.example.com/msp/keystore/"
	tlsCertPath   = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint  = "localhost:7051"
	gatewayPeer   = "peer0.org1.example.com"
	channelName   = "mychannel"
	chaincodeName = "basic"
	*/
	mspID         = "Org2MSP"
	cryptoPath    = "../../../../test-network/organizations/peerOrganizations/org2.example.com"
	certPath      = cryptoPath + "/users/User1@org2.example.com/msp/signcerts/cert.pem"
	keyPath       = cryptoPath + "/users/User1@org2.example.com/msp/keystore/"
	tlsCertPath   = cryptoPath + "/peers/peer0.org2.example.com/tls/ca.crt"
	peerEndpoint  = "localhost:9051"
	gatewayPeer   = "peer0.org2.example.com"
	channelName   = "mychannel"
	chaincodeName = "basic"
)


type Input struct {
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	User            string    `json:"user"`
	Amount float64 `json:"amount"`
	Category string `json:"category"`
	Timestamp int64 `json:"timestamp"`
}

type EndInput struct {
	ID string `json:"ID"`
	Amount float64 `json:"Amount"`
	Time int64 `json:"Time"`
}

type Energy struct {
	DocType          string    `json:"DocType"`
	Amount float64 `json:"Amount"`
	BidAmount float64 `json:"BidAmount"`
	SoldAmount float64 `json:"SoldAmount"`
	UnitPrice        float64   `json:"Unit Price"`
	BidPrice         float64   `json:"Bid Price"`
	GeneratedTime    int64 `json:"Generated Time"`
	BidTime          int64 `json:"Bid Time"`
	ID               string    `json:"ID"`
	EnergyID string `json:"EnergyID"`
	LargeCategory    string    `json:"LargeCategory"`
	Latitude         float64   `json:"Latitude"`
	Longitude        float64   `json:"Longitude"`
	Owner            string    `json:"Owner"`
	Producer         string    `json:"Producer"`
	Priority float64 `json:"Priority"`
	SmallCategory    string    `json:"SmallCategory"`
	Status           string    `json:"Status"`
	Distance float64 `json:"Distance"`
}



var StartTime int64
var EndTime int64

func main() {
	startHour := 6
	StartTime = time.Date(2015, time.March, 27, startHour, 0, 0, 0, time.Local).UnixNano()
	EndTime = time.Date(2015, time.March, 27, startHour + 24, 0, 0, 0, time.Local).UnixNano()
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

	bidTokens := getBidTokens(contract)
	energyTokens := getEnergyTokens(contract)

	fmt.Println(len(bidTokens))
	fmt.Println(len(energyTokens))

	DbResister(bidTokens, energyTokens)



	fmt.Printf("Token Resister end\n")

}

func DbResister(bidTokens []Energy, energyTokens []Energy) {
	db, err :=  sql.Open("sqlite3", "db/test1.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS "EnergyData" ("ID" TEXT PRIMARY KEY, "Producer" TEXT, "Latitude" REAL, "Longitude" REAL, "LargeCategory" TEXT, 
	"SmallCategory" TEXT, "Status" TEXT, "Amount" REAL, "SoldAmount" REAL, "GeneratedTime" INTEGER, "UnitPrice" REAL)`)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS "BidData" ("ID" TEXT PRIMARY KEY, "EnergyID" TEXT, "Producer" TEXT, "Owner" TEXT, 
	"LargeCategory" TEXT, "SmallCategory" TEXT, "Status" TEXT, "Amount" REAL, "BidAmount" REAL, "UnitPrice" REAL, "BidPrice" REAL, 
	"Priority" REAL, "BidTime" INTEGER, "Distance", REAL)`)
	if err != nil {
		panic(err)
	}

	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	stmt, err := tx.Prepare(`INSERT INTO EnergyData (ID, Producer, Latitude, Longitude, LargeCategory, SmallCategory, Status, Amount, 
		SoldAmount, GeneratedTime, UnitPrice) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	for _, energy := range energyTokens {
		_, err := stmt.Exec(energy.ID, energy.Producer, energy.Latitude, energy.Longitude, energy.LargeCategory, energy.SmallCategory, 
		energy.Status, energy.Amount, energy.SoldAmount, energy.GeneratedTime, energy.UnitPrice)
		if err != nil {
			panic(err)
		}
	}


	stmt, err = tx.Prepare(`INSERT INTO BidData (ID, EnergyID, Producer, Owner, LargeCategory, SmallCategory, Status, Amount, BidAmount, 
		UnitPrice, BidPrice, Priority, BidTime, Distance) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		panic(err)
	}

	for _, bid := range bidTokens {
		_, err := stmt.Exec(bid.ID, bid.EnergyID, bid.Producer, bid.Owner, bid.LargeCategory, bid.SmallCategory, bid.Status, bid.Amount, 
		bid.BidAmount, bid.UnitPrice, bid.BidPrice, bid.Priority, bid.BidTime, bid.Distance)
		if err != nil {
			panic(err)
		}
	}

	tx.Commit()

	//tx.Rollback()

	dcount := db.QueryRow(
		`SELECT count(*) FROM EnergyData`,
	)
	var eCount int
	err = dcount.Scan(&eCount)
	if (err != nil) {
		fmt.Println(err)
	}
	fmt.Printf("energy token num: %v", eCount)

	dcount = db.QueryRow(
		`SELECT count(*) FROM BidData`,
	)
	var bCount int
	err = dcount.Scan(&bCount)
	if (err != nil) {
		fmt.Println(err)
	}
	fmt.Printf("bid token num: %v", bCount)


	/*rows, err := db.Query(
		`SELECT * FROM Data`,
	)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var d Data
		err := rows.Scan(&d.ID, &d.UserName, &d.Latitude, &d.Longitude, &d.TotalAmountWanted, &d.FirstBidTime, &d.LastBidTime, &d.BatteryLife, 
			&d.Requested, &d.BidAmount, &d.BidSolar, &d.BidWind, &d.BidThermal, &d.GetAmount, &d.GetSolar, &d.GetWind, &d.GetThermal)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println(d)
	}*/
}
func getBidTokens(contract  *client.Contract) []Energy {
	var fullTokenList []Energy
	diff := EndTime - StartTime
	var interval int64 = 60 * 60 * 1000000000 // 1hour
	var i int64
	for i = 0; int64(i) < diff / interval; i++ {
		start := StartTime + i * interval
		end := StartTime + (i + 1) * interval - 1
		tokenList, b := queryBid(contract, start, end)
		if b == false {
			tokenList = []Energy{}
			var split int64 = 2
			var count int64 = 0
			for {
				splitStart := start + count * interval / split
				splitEnd := start + (count + 1) * interval / split - 1
				temp, b := queryBid(contract, splitStart, splitEnd)
				if b == false {
					tokenList = []Energy{}
					nextSplit := split + 1
					for {
						if (interval % nextSplit == 0) {
							break
						} else {
							nextSplit++
						}
					}
					split = nextSplit
					count = 0
				} else {
					tokenList = append(tokenList, temp...)
					count++
					if split == count {break}
				}
	
			}
		}
		fullTokenList = append(fullTokenList, tokenList...)
		
	}
	return fullTokenList

}

func getEnergyTokens(contract  *client.Contract) []Energy {
	var fullTokenList []Energy
	diff := EndTime - StartTime
	var interval int64 = 60 * 60 * 1000000000 // 1hour
	var i int64

	for i = 0; i < diff / interval; i++ {
		start := StartTime + i * interval
		end := StartTime + (i + 1) * interval - 1
		tokenList, b := queryByTime(contract, start, end)
		if b == false {
			tokenList = []Energy{}
			var split int64 = 2
			var count int64 = 0
			for {
				splitStart := start + count * interval / split
				splitEnd := start + (count + 1) * interval / split - 1
				temp, b := queryByTime(contract, splitStart, splitEnd)
				if b == false {
					tokenList = []Energy{}
					nextSplit := split + 1
					for {
						if (interval % nextSplit == 0) {
							break
						} else {
							nextSplit++
						}
					}
					split = nextSplit
					count = 0
				} else {
					tokenList = append(tokenList, temp...)
					count++
					if split == count {break}
				}
	
			}
		}
		fullTokenList = append(fullTokenList, tokenList...)
		
	}
	return fullTokenList

}

func queryBid(contract *client.Contract, startTime int64, endTime int64) ([]Energy, bool) {
	var tokens []Energy
	sStartTime := fmt.Sprintf("%v", startTime)
	sEndTime := fmt.Sprintf("%v", endTime)
	queryLoop:
	for {
		evaluateResult, err := contract.EvaluateTransaction("QueryBid", "success", sStartTime, sEndTime)
		if err != nil {
			log.Printf("query error:%v", err.Error())
		} else {
			if (len(evaluateResult) == 0) {
				return tokens, true
			}
			err = json.Unmarshal(evaluateResult, &tokens)
			if err != nil {
				log.Printf("unmarshal error: %v", err.Error())
			} else {
				if (len(tokens) > 99999) {
					return tokens, false
				}
				break queryLoop
			}
		}
	}
	return tokens, true
}

func queryByTime(contract *client.Contract, startTime int64, endTime int64) ([]Energy, bool) {
	var tokens []Energy
	sStartTime := fmt.Sprintf("%v", startTime)
	sEndTime := fmt.Sprintf("%v", endTime)
	queryLoop:
	for {
		evaluateResult, err := contract.EvaluateTransaction("QueryByTime", sStartTime, sEndTime)
		if err != nil {
			log.Printf("query error:%v", err.Error())
		} else {
			if (len(evaluateResult) == 0) {
				return tokens, true
			}
			err = json.Unmarshal(evaluateResult, &tokens)
			if err != nil {
				log.Printf("unmarshal error: %v", err.Error())
			} else {
				if (len(tokens) > 99999) {
					return tokens, false
				}
				break queryLoop
			}
		}
	}
	return tokens, true
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