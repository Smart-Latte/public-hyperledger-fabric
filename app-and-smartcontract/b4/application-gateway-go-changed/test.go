package main

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path"
	"time"
	"log"
	"sort"
	"strings"
	//"strconv"

	"encoding/json"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	mspID         = "Org1MSP"
	cryptoPath    = "../../../../test-network/organizations/peerOrganizations/org1.example.com"
	certPath      = cryptoPath + "/users/User1@org1.example.com/msp/signcerts/cert.pem"
	keyPath       = cryptoPath + "/users/User1@org1.example.com/msp/keystore/"
	tlsCertPath   = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint  = "localhost:7051"
	gatewayPeer   = "peer0.org1.example.com"
	channelName   = "mychannel"
	chaincodeName = "basic"
)
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
}


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

type BidReturn struct {
	ID string `json:"ID"`
	Message string `json:"Message"`
	Error error `json:"Error"`
}

const (
	dayNum = 2
	hourNum = 24
)

var StartTime int64
var EndTime int64
var Diff int64
var Speed int64
var Interval int64
var TokenLife int64
var StartHour int
var SolarOutput [dayNum][hourNum]float64
var WindOutput [dayNum][hourNum] float64
var SeaWindOutput [dayNum][hourNum] float64

func main() {
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

	/*energy := Energy{ID: "energy1", Producer: "Latte"}
	energies := []Energy{
		{ID: "bid1", Owner: "Coco"}, 
		{ID: "bid2", Owner: "Hanako"},
	}*/
	
	/*message, err := auctionEnd(contract, energy, energies)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(message)
	}*/

	Init(contract)
	energy, err := createEnergyToken(contract)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(energy)

	energy2, err := createEnergyToken2(contract)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(energy2)

	b, err := bidOk(contract)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(b)

	energy.DocType = "bid"
	energy.EnergyID = energy.ID
	energy.ID = "bid01"
	energy.Owner = "mayuko"
	energy.Priority = 0.5
	energy.BidAmount = 1000
	energy.BidPrice = 0.04
	energy.BidTime = 1427402500

	energy2.DocType = "bid"
	energy2.EnergyID = energy2.ID
	energy2.ID = "bid02"
	energy2.Owner = "mayuko"
	energy2.Priority = 0.5
	energy2.BidAmount = 500
	energy2.BidPrice = 0.04
	energy2.BidTime = 1427402500

	var energies []Energy
	energies = append(energies, energy)
	energies = append(energies, energy2)
	out, errList := bidOnEnergy2(contract, energies)
	if len(errList) != 0 {
		fmt.Println(err)
	} else {
		for _, o := range out {
			fmt.Println(o)
		}
	}

	renergy1, _ := readToken(contract ,"test01")
	fmt.Println(renergy1)
	renergy2, _ := readToken(contract, "test02")
	fmt.Println(renergy2)
	bid1, err := readToken(contract, "bid01")
	if (err != nil ) {
		fmt.Println(err)
	} else {
		fmt.Println(bid1)
	}
	bid2, err := readToken(contract, "bid02")
	if (err != nil ) {
		fmt.Println(err)
	} else {
		fmt.Println(bid2)
	}
	b1, err := bidOk(contract)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(b1)

	b2, err := bidOk2(contract)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(b2)

	/*fmt.Println(energy)
	bidOnEnergy(contract, "bid01", 0.026, "user1", 0.09, 100, "1427402500")
	bidOnEnergy(contract, "bid02", 0.025, "user1", 0.09, 1000, "1427402500")

	bidList, err := auctionEndQuery(contract, "test01", 1427402560)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(bidList)

	energyInput := EndInput{ID: "test01", Amount: 1000, Time: 1427402600}
	var bidInput []EndInput
	bidInput = append(bidInput, EndInput{ID: "bid01", Amount:100})
	bidInput = append(bidInput, EndInput{ID: "bid02", Amount:900})

	message, err := auctionEndTransaction(contract, energyInput, bidInput)
	if (err != nil) {
		fmt.Println(err.Error())
	}
	fmt.Println(message)*/


	/*
	message, err := auctionEndTransaction(contract, "test01", "1427402532")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println(message)
	}*/
	/*energy1, err := readToken(contract, "test01")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println(energy1)
	}
	bid1, err := readToken(contract, "bid1")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println(bid1)
	}
	bid2, err := readToken(contract, "bid2")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println(bid2)
	}*/
	
	

}

func bidOk(contract *client.Contract) (string, error){
	sBidPrice := fmt.Sprintf("%v", 0.03)
	sPriority := fmt.Sprintf("%v", 0.09)
	//isOk := true
	var result string
	fmt.Println(sBidPrice)
	fmt.Println(sPriority)

	/*	if ((time.Now().Unix() -Diff - StartTime) * Speed + StartTime > EndTime) {
			return false, fmt.Errorf("time up")
		}*/
		evaluateResult, err := contract.SubmitTransaction("BidOk", "test01", sBidPrice, sPriority)
		if err != nil {
			log.Printf("bid ok error: %v\n", err.Error())
		} else {
			result = string(evaluateResult)
			/*isOk, err = strconv.ParseBool(result)
			if err != nil {
				log.Printf("parse string to bool error:%v\n", err.Error())
			}*/
		}

	return result, nil
}
func bidOk2(contract *client.Contract) (string, error){
	sBidPrice := fmt.Sprintf("%v", 0.03)
	sPriority := fmt.Sprintf("%v", 0.09)
	//isOk := true
	var result string

	fmt.Println(sBidPrice)
	fmt.Println(sPriority)

		/*if ((time.Now().Unix() -Diff - StartTime) * Speed + StartTime > EndTime) {
			return false, fmt.Errorf("time up")
		}*/
		evaluateResult, err := contract.SubmitTransaction("BidOk", "test02", sBidPrice, sPriority)
		if err != nil {
			log.Printf("bid ok error: %v\n", err.Error())
		} else {
			result = string(evaluateResult)
			/*isOk, err = strconv.ParseBool(result)
			if err != nil {
				log.Printf("parse string to bool error:%v\n", err.Error())
			}*/
		}

	return result, nil
}

func Init(contract *client.Contract) {
	fmt.Printf("Submit Transaction: InitLedger, function creates the initial set of assets on the ledger \n")

	_, err := contract.SubmitTransaction("InitLedger")
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

func createEnergyToken(contract *client.Contract) (Energy, error) {
	var energy Energy
	evaluateResult, err := contract.SubmitTransaction("CreateEnergyToken", "test01", "40", "140", "producer", "1000", "Green", "solar", "1427402472")
	if err != nil {
		return energy, err
	}
	err = json.Unmarshal(evaluateResult, &energy)
	if err != nil {
		return energy, err
	}
	return energy, nil

}

func createEnergyToken2(contract *client.Contract) (Energy, error) {
	var energy Energy
	evaluateResult, err := contract.SubmitTransaction("CreateEnergyToken", "test02", "40", "140", "producer", "1000", "Green", "solar", "1427402475")
	if err != nil {
		return energy, err
	}
	err = json.Unmarshal(evaluateResult, &energy)
	if err != nil {
		return energy, err
	}
	return energy, nil

}

func bidOnEnergy2(contract *client.Contract, energies []Energy) ([]string, []string) {
	var errList []string
	var messageList []string
	energyJSON, err := json.Marshal(energies)
	if err != nil {
		panic(err)
	}
	//var out []BidReturn
	evaluateResult, err := contract.SubmitTransaction("BidOnEnergy", string(energyJSON))
	if err != nil {
		errList = strings.Split(err.Error(), ",")
		fmt.Println(err.Error())
	} else {
		message := string(evaluateResult)
		messageList = strings.Split(message, ",")
	}
	return messageList, errList
}

func bidOnEnergy(contract *client.Contract, bidId string, bidPrice float64, username string, batteryLife float64, amount float64, timestamp string) {
	//fmt.Printf("Evaluate Transaction: BidOnToken, function returns asset attributes\n")

	var message string
	sBidPrice := fmt.Sprintf("%v", bidPrice)
	sPriority := fmt.Sprintf("%v", 1 - batteryLife)
	sAmount := fmt.Sprintf("%v", amount)

	// create id

	evaluateResult, err := contract.SubmitTransaction("BidOnEnergy", bidId, "test01", username, sBidPrice, sPriority, sAmount, timestamp, "40", "140", "0.025")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		message = string(evaluateResult)
		fmt.Println(message)
	}
	return
}

func readToken(contract *client.Contract, id string) (Energy, error) {
	var energy Energy
	fmt.Printf("Async Submit Transaction: ReadToken: %s\n", id)

	evaluateResult, err := contract.EvaluateTransaction("readToken", id)
	if err != nil {
		return energy, err
				// panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	} else {
		err = json.Unmarshal(evaluateResult, &energy)
		if(err != nil) {
			return energy, err
		} else {
			fmt.Printf("%s success\n", id)
		}
	} 

	return energy, nil
}

func auctionEndQuery(contract *client.Contract, energyId string, timestamp int64) ([]Energy, error) {
	var bidList []Energy
	sTimestamp := fmt.Sprintf("%v", timestamp)

	queryLoop:
	for {
		/*if ((time.Now().Unix() -Diff - StartTime) * Speed + StartTime > EndTime) {
			return bidList, fmt.Errorf("time up")
		}*/
		evaluateResult, err := contract.SubmitTransaction("AuctionEndQuery", energyId, sTimestamp)
		if err != nil {
			log.Printf("auction end query error: %v\n", err.Error())
		} else {
			fmt.Printf("result length:%v\n", len(evaluateResult))
			if (len(evaluateResult) == 0) {
				return bidList, nil
			}
			err = json.Unmarshal(evaluateResult, &bidList)
			if err != nil {
				log.Printf("unmarshal error: %v\n", err.Error())
			}
			sort.SliceStable(bidList, func(i, j int) bool {
				return bidList[i].BidTime < bidList[j].BidTime
			})
		
			sort.SliceStable(bidList, func(i, j int) bool {
				return bidList[i].Priority > bidList[j].Priority
			})
		
			sort.SliceStable(bidList, func(i, j int) bool {
				return bidList[i].BidPrice > bidList[j].BidPrice
			})
			break queryLoop
		}
	}
	
	return bidList, nil

}

func auctionEndTransaction(contract *client.Contract, energyInput EndInput, bidInput []EndInput) (string, error){
	var message string
	energyJSON, err := json.Marshal(energyInput)
	if err != nil {
		panic(err)
	}
	bidJSON, err := json.Marshal(bidInput)
	if err != nil {
		panic(err)
	}

	for {
		if ((time.Now().Unix() -Diff - StartTime) * Speed + StartTime > EndTime) {
			return "", fmt.Errorf("time up")
		}
		evaluateResult, err := contract.SubmitTransaction("AuctionEnd", string(energyJSON), string(bidJSON))
		if err != nil {
			log.Printf("producer auction end error: %v\n", err.Error())
		} else {
			message = string(evaluateResult)
			break
		}
	}
	return message, nil
}


/*func auctionEndTransaction(contract *client.Contract, energyInput EndInput, bidInput []EndInput) (string, error){
	var message string
	energyJSON, err := json.Marshal(energyInput)
	if err != nil {
		panic(err)
	}
	bidJSON, err := json.Marshal(bidInput)
	if err != nil {
		panic(err)
	}

	return message, nil
	for {
		if ((time.Now().Unix() -Diff - StartTime) * Speed + StartTime > EndTime) {
			return "", fmt.Errorf("time up")
		}
		evaluateResult, err := contract.SubmitTransaction("AuctionEnd", string(energyJSON), string(bidJSON))
		if err != nil {
			log.Printf("prudcer auction end error: %v\n", err.Error())
		} else {
			message = string(evaluateResult)
			break
		}
	}
	return message, nil
}*/

func auctionEnd(contract *client.Contract, energy Energy, energies []Energy) (string, error) {
	energyJSON, err := json.Marshal(energy)
	if err != nil {
		panic(err)
	}
	energiesJSON, err := json.Marshal(energies)
	if err != nil {
		panic(err)
	}
	evaluateResult, err := contract.SubmitTransaction("AuctionEnd", string(energyJSON), string(energiesJSON))
	if err != nil {
		return "", err
	}
	fmt.Printf("evaluateResult length : %v\n", len(evaluateResult))
	message := string(evaluateResult)

	// fmt.Printf("*** %s Result:%s\n", energyId, massage)
	return message, nil
	
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
