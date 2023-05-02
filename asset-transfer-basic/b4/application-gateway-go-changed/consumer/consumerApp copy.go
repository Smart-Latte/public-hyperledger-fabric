package consumer

import (
	"fmt"
	"time"
	"math/rand"
	"github.com/hyperledger/fabric-gateway/pkg/client"
)

type Input struct {
	UserName string
	Latitude float64
	Longitude float64
	Battery float64
	Charged float64
	ChargeTime float64
	Amount float64
	BatteryLife float64
}
/*
type ResultInput struct {
	UserName string
	StartTime int64
	EndTime int64
}*/

func ConsumerHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("consumerHandler")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed) //405
		w.Write([]byte("Only POST"))
		return
	}
	if r.Header.Get("Content-Type") != "application/json; charset=utf-8" {
		w.WriteHeader(http.StatusBadRequest) //400
		w.Write([]byte("Only json"))
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) //400
		w.Write([]byte(err.Error()))
		return
	}
	var requestInput Input
	err = json.Unmarshal(body, &requestInput)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError) //500
		fmt.Println("Unmarshal error")
		fmt.Println(err.Error())
		w.Write([]byte(err.Error()))
		return
	}
	go RealConsume()
	fmt.Fprintf(w, "accept")
}

// 充電開始時間(差分)、バッテリー容量(Wh)、チャージ済み(Wh)、充電時間(hour)、最終的なバッテリー残量(0から1)
func RealConsume(input Input) {
	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	mspID         := "Org2MSP"
	cryptoPath    := "../../../../test-network/organizations/peerOrganizations/org2.example.com"
	certPath      := cryptoPath + "/users/User1@org2.example.com/msp/signcerts/cert.pem"
	keyPath      := cryptoPath + "/users/User1@org2.example.com/msp/keystore/"
	tlsCertPath   := cryptoPath + "/peers/9051.org2.example.com/tls/ca.crt"
	peerEndpoint  := "localhost:9051"
	gatewayPeer   := "peer0.org2.example.com"
	channelName   := "mychannel"
	chaincodeName := "basic"
	
	clientConnection := newGrpcConnection(tlsCertPath, gatewayPeer, peerEndpoint)
	defer clientConnection.Close()
	
	id := newIdentity(certPath, mspID)
	sign := newSign(keyPath)

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

	endTimer := time.NewTimer(time.Duration((EndTime - ((time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime)) / Speed) * time.Nanosecond)
	
	consumeData := []Data{}
	//input := Input{UserName: username, Latitude: lat, Longitude: lon}
	chargedEnergy := charged
	finalBattery := battery * finalLife
	require := finalBattery - charged
	chargePerSec := battery / chargeTime / 60 / 60
	requestMax := battery * finalLife / chargeTime / (60 / float64(TokenLife))
	//fmt.Println(requestAmount)
	var beforeUse float64 = 0
	var err error
	highLow := 0

	zeroCount := 0
	loop:
		for i := 0; ; i++ {

			var requestAmount float64
			if (chargedEnergy >= finalBattery) {
				// fmt.Println("charged")
				endTimer.Stop()
				// http finish
				break loop
			}
			if (requestMax < finalBattery - chargedEnergy) {
				requestAmount = requestMax - beforeUse
			} else {
				requestAmount = finalBattery - chargedEnergy
			}

			batteryLife := chargedEnergy / battery
			consumeData = append(consumeData, Data{UserName: username, BatteryLife: batteryLife, HighPrice: highLow, Requested: requestAmount, Latitude: lat, Longitude: lon, TotalAmountWanted: require})
			var bidEnergies []Energy

			select {
			case <- endTimer.C:
				//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
				//fmt.Printf("CONSUMER END TIMER new: %v\n", time.Unix(0, timestamp))
				// http timeup
				break loop
			default:
				bidEnergies, consumeData[i], err = Bid(contract, consumeData[i])
				if err != nil {
					//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
					//fmt.Printf("CONSUMER END TIMER new: %v\n", time.Unix(0, timestamp))
					// http timeup
					break loop
				}
				// fmt.Printf("%s bid energy: %vWh\n", username, consumeData[i].BidAmount)
			}

		// 	fmt.Println("next is result")
			if (consumeData[i].BidAmount == 0) {
				zeroCount++
				highLow = 1
				
				if (zeroCount > 10) {
					//fmt.Println("%s long NO BID", username)
					longZeroTimer := time.NewTimer(30 * 60 * 1000000000 * time.Nanosecond / time.Duration(Speed))
					// http longZero
					select {
						case <- endTimer.C:
							//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
							//fmt.Printf("CONSUMER END TIMER: %v\n", time.Unix(0, timestamp))
							// http timeup
							break loop
						case <- longZeroTimer.C:
							zeroCount = 0
							continue loop
					}
				} else {
					//fmt.Printf("%s, nothing, zeroCount:%v\n", username, zeroCount)
					nothingTimer := time.NewTimer(60 * 1000000000 * time.Nanosecond / time.Duration(Speed))
					// http zero
					select {
					case <- endTimer.C:
						//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
						//fmt.Printf("CONSUMER END TIMER: %v\n", time.Unix(0, timestamp))
						break loop
					case <- nothingTimer.C:
						continue loop
					}
				}
			} else {
				highLow = 0
				zeroCount = 0
				// http bid success
			}

			checkTime := (consumeData[i].LastBidTime + (Interval * 60 + 30) * 1000000000)
			wait := (checkTime - ((time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime)) / Speed

			//fmt.Printf("%s check wait:%v sec\n", username, float64(wait) / 1000000000)

			if (wait > 0) {
				resultTimer := time.NewTimer(time.Duration(wait) * time.Nanosecond)
				select {
				case <- resultTimer.C:
					// fmt.Println("result Timer")
				case <- endTimer.C:
					//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
					//fmt.Printf("CONSUMER END TIMER: %v\n", time.Unix(0, timestamp))
					// http timeup
					break loop
				}
			}
			// bidResult
			select{
			case <-endTimer.C:
				//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
				//fmt.Printf("CONSUMER END TIMER: %v\n", time.Unix(0, timestamp))
				// http timeup
				break loop
			default:
				consumeData[i], err = BidResult(contract, bidEnergies, consumeData[i])
				if err != nil {
					// http timeup
					//fmt.Printf("time up: %v\n", err)
					break loop
				} else {
					// http get energy amount
					//log.Printf("%s: count: %d, bidEnergy:%g, getEnergy:%gWh\n", username, i, consumeData[i].BidAmount, consumeData[i].GetAmount)
				}
			}

			chargedEnergy += consumeData[i].GetAmount
			var nextBidWait float64
			if (consumeData[i].GetAmount >= chargePerSec * 60 * 5) {
				nextBidWait = ((consumeData[i].GetAmount + beforeUse) / chargePerSec) - 5 * 60
				beforeUse = chargePerSec * 60 * 5
			} else {
				nextBidWait = 0
			}
			nextBidTimer := time.NewTimer(time.Duration(nextBidWait * 1000000000) * time.Nanosecond / time.Duration(Speed))
			//fmt.Printf("%s next bid: after %v min\n", username, nextBidWait / 60 / float64(Speed))

			if consumeData[i].GetAmount == 0 {
				highLow = 1
			} else {
				highLow = 0
			}

			select {
			case <- nextBidTimer.C:
				break
			case <- endTimer.C:
				nextBidTimer.Stop()
				//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
				//fmt.Printf("CONSUMER END TIMER: %v\n", time.Unix(0, timestamp))
				break loop
			}
		}
	//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
	//fmt.Printf("consumer %s finish start:%v, end:%v, finalBattery:%v, chargedEnergy:%v\n", username, time.Unix(0, consumerStart), time.Unix(0, timestamp), finalBattery, chargedEnergy)
	return consumeData

}
