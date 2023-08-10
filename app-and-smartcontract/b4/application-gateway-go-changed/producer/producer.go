package producer

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path"
	"time"
	"sync"
	"math"

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
var BigSolarOutput [dayNum][hourNum]float64
var HouseSolarOutput [dayNum][hourNum]float64
var WindSpeed [dayNum][hourNum] float64
var SeaWindSpeed [dayNum][hourNum] float64

func AllProducers(start int64, end int64, difference int64, mySpeed int64, auctionInterval int64, life int64, 
	bigSOutput [dayNum][hourNum]float64, houseSoutput [dayNum][hourNum]float64, wSpeed [dayNum][hourNum]float64, 
	swSpeed [dayNum][hourNum]float64, hour int) {
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

	StartTime = start
	EndTime = end
	Diff = difference
	Speed = mySpeed
	Interval = auctionInterval
	TokenLife = life
	StartHour = hour
	BigSolarOutput = bigSOutput
	HouseSolarOutput = houseSoutput
	WindSpeed = wSpeed
	SeaWindSpeed = swSpeed

	var solarOut float64 = 1000000
	var realWindOut float64 = 1990000
	var dummySolar float64 = 40000
	var dummyWind float64 = 11000
	var rating float64 = 12.5
	var cutIn float64 = 2.5
	var thermalOutput[dayNum][hourNum] float64
	var seaWindOutput[dayNum][hourNum] float64

	for i := 0; i < dayNum; i++ {
		for j := 0; j < hourNum; j++ {
			solar := solarOut * BigSolarOutput[i][j]
			dSolar := dummySolar * HouseSolarOutput[i][j] * 4
			var seaWind float64
			var dWind float64
			if SeaWindSpeed[i][j] >= cutIn {
				seaWindOutput[i][j] = realWindOut * math.Pow((SeaWindSpeed[i][j] / rating), 3)
				seaWind = seaWindOutput[i][j] * 3
			}
			if WindSpeed[i][j] >= cutIn {
				dWind = dummyWind * math.Pow((WindSpeed[i][j] / rating), 3) * 4
			}
			out := solar + dSolar + seaWind + dWind // グリーンエネルギーの最大出力
			var maxOut float64
			if (i == 0 && j < StartHour + 8) {
				maxOut = 420 / 24.0 * 5000 * float64(j - (StartHour - 1)) + 210 / 12.0 * 32000
			} else if (i == 0 && j < 18 || i == 1 && j > 5) {
				maxOut = 420 / 24.0 * 5000 * 8 + 210 / 12.0 * 32000
			} else {
				maxOut = 420 / 24.0 * 5000 * 8
			}
			/*
			if (i == 0 && j < 18) {
				maxOut = 1610000
				//thermalOutput[i][j] = 1
			} else {
				maxOut = 1050000
				//thermalOutput[i][j] = 1
			}*/
			if (maxOut < out) {
				con := out - maxOut
				seaWindOutput[i][j] = (seaWind - con) / 3
				if seaWindOutput[i][j] < 0 {
					solar += seaWind - con
					seaWindOutput[i][j] = 0
				}
			}
			thermalOutput[i][j] = maxOut - out
			if (thermalOutput[i][j] < 0) {
				thermalOutput[i][j] = 0
				//thermalOutput[i][j] = 1
			}
			BigSolarOutput[i][j] = solar
			fmt.Printf("day:%v, hour:%v, solar:%v, dSolar:%v, wind:%v, dWind:%v, thermal:%v\n", i, j, solar, dSolar, seaWindOutput[i][j] * 3, dWind, thermalOutput[i][j])
		}
	}


	var wg sync.WaitGroup

	wg.Add(5)
	go func() {
		defer wg.Done()
		Produce(contract, "real-solar-producer0", 40.2297629645958, 140.010266575019, "solar", 1, BigSolarOutput, 0)
	}()
	go func() {
		defer wg.Done()
		// SeaWindProducer(contract, "real-wind-producer0", 40.2160279724715, 140.002846271612, "wind", realWindOut, rating, cutIn, SeaWindSpeed, 1)
		Produce(contract, "real-wind-producer0", 40.2160279724715, 140.002846271612, "wind", 1, seaWindOutput, 1)
	}()
	go func() {
		defer wg.Done()
		//SeaWindProducer(contract, "real-wind-producer1", 40.2095028757269, 139.997337258476, "wind", realWindOut, rating, cutIn, SeaWindSpeed, 2)
		Produce(contract, "real-wind-producer1", 40.2095028757269, 139.997337258476, "wind", 1, seaWindOutput, 2)
	}()
	go func() {
		defer wg.Done()
		// SeaWindProducer(contract, "real-wind-producer2", 40.2021377588529, 140.068615482843, "wind", realWindOut, rating, cutIn, SeaWindSpeed, 3)
		Produce(contract, "real-wind-producer2", 40.2021377588529, 140.068615482843, "wind", 1, seaWindOutput, 3)
	}()
	go func() {
		defer wg.Done()
		Produce(contract, "real-thermal-producer0", 40.1932732666231, 139.992165531859, "thermal", 1, thermalOutput, 4)
	}()

	/*for i := 0; i < 4; i++ { // 4
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			DummySolarProducer(contract, fmt.Sprintf("solarProducerGroup%d", n), 40.17463042136363, 40.2330209148308, 140.010266575019, 140.068615482843, "solar", dummySolar, SolarOutput, int64(n + 10000))
			//DummySolarProducer(contract, fmt.Sprintf("solarProducerGroup%d", n), 40.17463042136363, 40.2330209148308, 139.992165531859, 140.068615482843, "solar", dummySolar, SolarOutput, int64(n + 10000))
			// DummySolarProducer(contract, fmt.Sprintf("solarProducer%d", n), 40.1932732666231, 40.2297629645958, 139.992165531859, 140.068615482843, "solar", 4000, SolarOutput, int64(n + 10000))

		}(i)
		go func(n int) {
			defer wg.Done()
			DummyWindProducer(contract, fmt.Sprintf("windProducerGroup%d", n), 40.17463042136363, 40.2330209148308, 140.010266575019, 140.068615482843, "wind", dummyWind, rating, cutIn, WindSpeed, int64(n + 1000))
			// DummyWindProducer(contract, fmt.Sprintf("windProducerGroup%d", n), 40.17463042136363, 40.2330209148308, 139.992165531859, 140.068615482843, "wind", dummyWind, rating, cutIn, WindSpeed, int64(n + 1000))
			// DummyWindProducer(contract, fmt.Sprintf("windProducer%d", n), 40.1932732666231, 40.2297629645958, 139.992165531859, 140.068615482843, "wind", 1100, 12.5, 2.5, WindOutput, int64(n + 1000))
		}(i)
	}*/
	wg.Add(8)
		go func() {
			defer wg.Done()
			DummySolarProducer(contract, "solarProducerGroup0", 40.17463042136363, 40.203825668097228, 140.010266575019, 140.039441028931, 
			"solar", dummySolar, HouseSolarOutput, int64(10000))
		}()
		go func() {
			defer wg.Done()
			DummySolarProducer(contract, "solarProducerGroup1", 40.17463042136363, 40.203825668097228, 140.039441028931, 140.068615482843, 
			"solar", dummySolar, HouseSolarOutput, int64(10001))
		}()
		go func() {
			defer wg.Done()
			DummySolarProducer(contract, "solarProducerGroup2", 40.20382566809722, 40.2330209148308, 140.010266575019, 140.039441028931, 
			"solar", dummySolar, HouseSolarOutput, int64(10002))
		}()
		go func() {
			defer wg.Done()
			DummySolarProducer(contract, "solarProducerGroup3", 40.20382566809722, 40.2330209148308, 140.039441028931, 140.068615482843, 
			"solar", dummySolar, HouseSolarOutput, int64(10003))
		}()
		go func() {
			defer wg.Done()
			DummyWindProducer(contract, "windProducerGroup0", 40.17463042136363, 40.203825668097228, 140.010266575019, 140.039441028931,
			"wind", dummyWind, rating, cutIn, WindSpeed, int64(1000))
		}()
		go func() {
			defer wg.Done()
			DummyWindProducer(contract, "windProducerGroup1", 40.17463042136363, 40.203825668097228, 140.039441028931, 140.068615482843, 
			"wind", dummyWind, rating, cutIn, WindSpeed, int64(1001))
		}()
		go func() {
			defer wg.Done()
			DummyWindProducer(contract, "windProducerGroup2", 40.20382566809722, 40.2330209148308, 140.010266575019, 140.039441028931, 
			"wind", dummyWind, rating, cutIn, WindSpeed, int64(1002))
		}()
		go func() {
			defer wg.Done()
			DummyWindProducer(contract, "windProducerGroup3", 40.20382566809722, 40.233020914838, 140.039441028931, 140.068615482843, 
			"wind", dummyWind, rating, cutIn, WindSpeed, int64(1003))
		}()

	wg.Wait()

	fmt.Printf("all producer end\n")

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
