/*
Copyright 2021 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
// 需要家
// Org2のユーザで実行
// input: requestedTokenNum, batteryLife

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"strconv"
	"math"
	"sort"
	"sync"
	
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-protos-go-apiv2/gateway"
	"google.golang.org/grpc/status"
)

// var assetId = fmt.Sprintf("energy%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6) 

type Energy struct {
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
	MyBidStatus		 string    `json:"My Bid Status"`
}

const (
	earthRadius = 6378137.0
	requestedTokenNum int = 10
	batteryLife = 10 //%
	pricePerMater = 0.01
	searchRange = (100 - batteryLife) * 0.1 * 1000 // 9km * 1000m
	myLatitude = 35.5552824466371 //0から89まで
	myLongitude = 139.65527497388206
	username = "user2"
	layout = "2006-01-02T15:04:00+09:00"
)

var tokenNum int = requestedTokenNum

func Start(contract *client.Contract) {

}

func Buy(contract *client.Contract) {
	// batteryLifeから検索範囲決定
	fmt.Printf("searchRange:%g\n", searchRange)

	lowerLat, upperLat, lowerLng, upperLng := determineRange(searchRange)
	energies := queryByLocationRange(contract, lowerLat, upperLat, lowerLng, upperLng)
	
	// fmt.Println(energies)
	fmt.Printf("length of energies: %d\n", len(energies))

	timestamp := time.Now()
	auctionStartTimeCompare := timestamp.Add(time.Minute * -5)

	validEnergies := []Energy{}

	for _, energy := range energies {
		distance := distance(myLatitude, myLongitude, energy.Latitude, energy.Longitude)
		if distance <= searchRange && auctionStartTimeCompare.After(energy.AuctionStartTime) == false {
			energy.BidPrice = energy.UnitPrice + distance * pricePerMater
			validEnergies = append(validEnergies, energy)
			fmt.Println("it's valid")
			fmt.Printf("id:%s, latitude:%g, longitude:%g, unitPrice:%g, distance:%g, bidPrice:%g\n", 
			energy.ID, energy.Latitude, energy.Longitude, energy.UnitPrice, distance, energy.BidPrice)
		}else {
			fmt.Println("it's invalid")
			fmt.Printf("id:%s, latitude: %g, longitude:%g, unitPrice:%g, distance:%g, auctionStartTime:%s\n",
		energy.ID, energy.Latitude, energy.Longitude, energy.UnitPrice, distance, energy.AuctionStartTime.Format(layout))
		}
		
	}

	sort.Slice(validEnergies, func(i, j int) bool {
        return validEnergies[i].BidPrice > validEnergies[j].BidPrice
    })
	//fmt.Println(validEnergies)

	// validEnergiesのうち、上からtokenNum個分Bid

	var bidNum int
	success := []Energy{}
	
	for {
		if(tokenNum == 0 || len(validEnergies) == 0) {
			break
		}
		fmt.Printf("requested token:%d\n", tokenNum)
		fmt.Printf("valid energy token:%d\n", len(validEnergies))
		if(tokenNum > len(validEnergies)){
			bidNum = len(validEnergies)
		}else {
			bidNum = tokenNum
		}
		fmt.Printf("max:%d\n", bidNum)

		tempSuccess := bid(contract, validEnergies, bidNum)

		/*for _, energy := range tempSuccess {
			success = append(success, energy)
		}*/
		success = append(success, tempSuccess...)
		// println(success)
		// tempの長さ分、validEnergyから引く
		validEnergies = validEnergies[bidNum:]
		tokenNum -= len(tempSuccess)
	}

	var wg sync.WaitGroup
	wg.Add(len(success))
	for i := 0; i < len(success) ; i++ {
		go func (i int) {
			defer wg.Done()
			auctionStartTime := success[i].AuctionStartTime
			auctionEndTime := auctionStartTime.Add(time.Minute * 5)
			fmt.Printf("id:%s, auctionEndTime:%s\n", success[i].ID, auctionEndTime.Format(layout))
			nowTime := time.Now()
			fmt.Println(nowTime)
			fmt.Println(auctionEndTime.Sub(nowTime))
			timer := time.NewTimer(auctionEndTime.Sub(nowTime))
			<-timer.C
			auctionEndToken := readToken(contract, success[i].ID)
			if (auctionEndToken.Owner == username) {
				// notice?
				fmt.Println("you are a winner.")
			} else {
				fmt.Println("you are a loser.")
			}
		}(i)
	}

	wg.Wait()
	
}

func readToken(contract *client.Contract, energyId string) Energy {
	fmt.Printf("Async Submit Transaction: ReadToken'\n")

	evaluateResult, err := contract.EvaluateTransaction("ReadToken", energyId)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	//result := formatJSON(evaluateResult)

	result := Energy{}

	err = json.Unmarshal(evaluateResult, &result)
	if (err != nil) {
		fmt.Printf("unmarshal error")
	}

	return result

	//fmt.Printf("*** Result:%s\n", result)
}

func bid(contract *client.Contract, energies []Energy, bidNum int) []Energy {
	successEnergy := []Energy{}
	//leftEnergy := energies
	for i := 0; i < bidNum; i++ {
		fmt.Printf("id:%s, auctionStartTime:%s\n",
		energies[i].ID, energies[i].AuctionStartTime.Format(layout))
		massage := bidOnToken(contract, energies[i].ID, energies[i].BidPrice)
		fmt.Println(massage)
		if (massage == "your bid was successful") {
			successEnergy = append(successEnergy, energies[i])
			// auctionstart + 5min 経ったら見に行く
		}
	}
	return successEnergy
}

func bidOnToken(contract *client.Contract, energyId string, bidPrice float64) (string) {
	//fmt.Printf("Evaluate Transaction: BidOnToken, function returns asset attributes\n")
	var timestamp = time.Now()
	var stringTimestamp = timestamp.Format(layout)
	var stringBidPrice = strconv.FormatFloat(bidPrice, 'f', -1, 64)
	//fmt.Printf("id:%s, timestamp:%s, price:%s\n", energyId, stringTimestamp, stringBidPrice)
	evaluateResult, err := contract.SubmitTransaction("BidOnToken", energyId, username, stringBidPrice, stringTimestamp)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	//result := formatJSON(evaluateResult)
	massage := string(evaluateResult)
	/* "your bid was successful" */
	return massage
}


func determineRange(length float64) (lowerLat float64, upperLat float64, lowerLng float64, upperLng float64) {
	// 緯度固定で経度求める
	rlat := myLatitude * math.Pi / 180
	r := length / earthRadius
	angle := math.Cos(r)

	lngTmp := (angle - math.Sin(rlat) * math.Sin(rlat)) / (math.Cos(rlat) * math.Cos(rlat))
	rlngDifference := math.Acos(lngTmp)
	lngDifference := rlngDifference * 180 / math.Pi
	returnLowerLng := myLongitude - lngDifference
	returnUpperLng := myLongitude + lngDifference

	// 経度固定で緯度求める
	// rlng := myLongitude * math.Pi / 180
	//latTmp := angle / (math.Sin(rlat) + math.Cos(rlat))
	rSinLat := math.Sin(rlat)
	rCosLat := math.Cos(rlat)
	square := math.Sqrt(math.Pow(rSinLat, 2) + math.Pow(rCosLat, 2))
	latTmp := math.Asin(angle / square)
	solutionRLat := latTmp - math.Acos(rSinLat / square)
	// 緯度はプラスなため、solutionLatは常にmylatitudeより小さい
	returnLowerLat := solutionRLat * 180 / math.Pi
	returnUpperLat := 2 * myLatitude - math.Abs(lowerLat) //緯度が0のとき、lowerLatがマイナスなため。日本は関係ないが。


	fmt.Printf("lowerLng:%g\n", returnLowerLat)
	fmt.Printf("uperLng:%g\n", returnUpperLat)
	fmt.Printf("lowerLng:%g\n", returnLowerLng)
	fmt.Printf("uperLng:%g\n", returnUpperLng)

	return returnLowerLat, returnUpperLat, returnLowerLng, returnUpperLng

}

func queryByLocationRange(contract *client.Contract, lowerLat float64, upperLat float64, lowerLng float64, upperLng float64) ([]Energy) {
	strLowerLat := strconv.FormatFloat(lowerLat, 'f', -1, 64)
	strUpperLat := strconv.FormatFloat(upperLat, 'f', -1, 64)
	strLowerLng := strconv.FormatFloat(lowerLng, 'f', -1, 64)
	strUpperLng := strconv.FormatFloat(upperLng, 'f', -1, 64)

	fmt.Printf("Async Submit Transaction: QueryByLocationRange'\n")

	evaluateResult, err := contract.EvaluateTransaction("QueryByLocationRange", "generated", strLowerLat, strUpperLat, strLowerLng, strUpperLng)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}

	fmt.Println(len(evaluateResult))
	result := []Energy{}

	err = json.Unmarshal(evaluateResult, &result)
	if(err != nil) {
		fmt.Printf("unmarshal error")
	}

	return result

}

func distance(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	// 緯度経度をラジアンに変換
	rlat1 := lat1 * math.Pi / 180
	rlng1 := lng1 * math.Pi / 180
	rlat2 := lat2 * math.Pi / 180
	rlng2 := lng2 * math.Pi / 180

	// 2点の中心角を求める。
	/*cos(c)=cos(a)cos(b) + sin(a)sin(b)cos(c)
	= cos(pi/2 - lat1)cos(pi/2 - lat2) + sin(lat1)sin(lat2)cos(lng1 - lng2)
	= cos(sin(lat1)sin(lat2) + sin(lat1)sin(lat2)cos(lng1 - lng2))
	*/
	angle := 
		math.Sin(rlat1) * math.Sin(rlat2) +
		math.Cos(rlat1) * math.Cos(rlat2) *
		math.Cos(rlng1 - rlng2)

	r := math.Acos(angle)
	distance := earthRadius * r
	
	return distance
}


// This type of transaction would typically only be run once by an application the first time it was started after its
// initial deployment. A new version of the chaincode deployed later would likely not need to run an "init" function.
func InitLedger(contract *client.Contract) {
	fmt.Printf("Submit Transaction: InitLedger, function creates the initial set of assets on the ledger \n")

	_, err := contract.SubmitTransaction("InitLedger")
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Evaluate a transaction to query ledger state.
func GetAllTokens(contract *client.Contract) {
	fmt.Println("Evaluate Transaction: GetAllTokens, function returns all the current assets on the ledger")

	evaluateResult, err := contract.EvaluateTransaction("GetAllTokens")
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}
/*
// Submit a transaction synchronously, blocking until it has been committed to the ledger.
func CreateToken(contract *client.Contract) {
	fmt.Printf("Submit Transaction: CreateToken, creates new token with ID, Latitude, Longitude, Owner, Large Category, Small Category and timestamp \n")
	var timestamp = time.Now()
	var layout = "2006-01-02T15:04:00Z"
	var stringTimestamp = timestamp.Format(layout)
	fmt.Printf("%s\n", stringTimestamp)
	result, err := contract.SubmitTransaction("CreateToken", assetId, "35", "170", "User2", "Green", "solor", stringTimestamp)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}
	//result :=  formatJSON(jsonResult)
	fmt.Printf("%s\n", result)
	fmt.Printf("*** Transaction committed successfully\n")
}*/

// Evaluate a transaction by assetID to query ledger state.
func BidOnToken(contract *client.Contract) {
	fmt.Printf("Evaluate Transaction: BidOnToken, function returns asset attributes\n")
	//var timestamp = time.Now()
	//var layout = "2006-01-02T15:04:00Z"
	//var stringTimestamp = timestamp.Format(layout)
	evaluateResult, err := contract.SubmitTransaction("BidOnToken", "1000000", "Mayuko", "0.02", "2022-11-13T23:23:00Z")
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	//result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", evaluateResult)
}
/*
func AuctionEnd(contract *client.Contract) {
	fmt.Printf("Evaluate Transaction: BidOnToken, function returns asset attributes\n")
	var timestamp = time.Now()
	var layout = "2006-01-02T15:04:00Z"
	var stringTimestamp = timestamp.Format(layout)
	evaluateResult, err := contract.SubmitTransaction("AuctionEnd", assetId, "Mayuko", stringTimestamp)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	//result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", evaluateResult)
}
*/
// Submit transaction asynchronously, blocking until the transaction has been sent to the orderer, and allowing
// this thread to process the chaincode response (e.g. update a UI) without waiting for the commit notification
func QueryByStatus(contract *client.Contract) {
	fmt.Printf("Async Submit Transaction: QueryByStatus'\n")

	evaluateResult, err := contract.EvaluateTransaction("QueryByStatus", "generated")
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

func QueryByLocationRange(contract *client.Contract) {
	fmt.Printf("Async Submit Transaction: QueryByLocationRange'\n")

	evaluateResult, err := contract.EvaluateTransaction("QueryByLocationRange", "generated", "35.54", "35.55", "139.67", "139.68")
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}
/*
func ReadToken(contract *client.Contract) {
	fmt.Printf("Async Submit Transaction: ReadToken, updates existing asset owner'\n")

	evaluateResult, err := contract.EvaluateTransaction("ReadToken", assetId)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}*/

// Submit transaction, passing in the wrong number of arguments ,expected to throw an error containing details of any error responses from the smart contract.
func ExampleErrorHandling(contract *client.Contract) {
	fmt.Println("Submit Transaction: UpdateAsset asset70, asset70 does not exist and should return an error")

	_, err := contract.SubmitTransaction("UpdateAsset", "energy4")
	if err != nil {
		switch err := err.(type) {
		case *client.EndorseError:
			fmt.Printf("Endorse error with gRPC status %v: %s\n", status.Code(err), err)
		case *client.SubmitError:
			fmt.Printf("Submit error with gRPC status %v: %s\n", status.Code(err), err)
		case *client.CommitStatusError:
			if errors.Is(err, context.DeadlineExceeded) {
				fmt.Printf("Timeout waiting for transaction %s commit status: %s", err.TransactionID, err)
			} else {
				fmt.Printf("Error obtaining commit status with gRPC status %v: %s\n", status.Code(err), err)
			}
		case *client.CommitError:
			fmt.Printf("Transaction %s failed to commit with status %d: %s\n", err.TransactionID, int32(err.Code), err)
		}

		// Any error that originates from a peer or orderer node external to the gateway will have its details
		// embedded within the gRPC status error. The following code shows how to extract that.
		statusErr := status.Convert(err)
		for _, detail := range statusErr.Details() {
			switch detail := detail.(type) {
			case *gateway.ErrorDetail:
				fmt.Printf("Error from endpoint: %s, mspId: %s, message: %s\n", detail.Address, detail.MspId, detail.Message)
			}
		}
	}
}

// Format JSON data
func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, " ", ""); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}
