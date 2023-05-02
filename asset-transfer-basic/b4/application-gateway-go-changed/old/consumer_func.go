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
	// "context"
	"encoding/json"
	// "errors"
	"fmt"
	"time"
	"strconv"
	"math"
	"sort"
	"sync"
	"net/http"
	
	"github.com/hyperledger/fabric-gateway/pkg/client"
	// "github.com/hyperledger/fabric-protos-go-apiv2/gateway"
	// "google.golang.org/grpc/status"
)

// var assetId = fmt.Sprintf("energy%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6) 

type Energy struct {
	DocType          string    `json:"DocType`
	UnitPrice        float64   `json:"Unit Price"`
	BidPrice         float64   `json:"Bid Price"`
	GeneratedTime    time.Time `json:"Generated Time"`
	AuctionStartTime time.Time `json:"Auction Start Time"`
	// BidTime          time.Time `json:"Bid Time"`
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

type BidResultEnergy struct {
	DocType          string    `json:"DocType`
	UnitPrice        float64   `json:"Unit Price"`
	BidPrice         float64   `json:"Bid Price"`
	GeneratedTime    time.Time `json:"Generated Time"`
	AuctionStartTime time.Time `json:"Auction Start Time"`
	// BidTime          time.Time `json:"Bid Time"`
	ID               string    `json:"ID"`
	LargeCategory    string    `json:"LargeCategory"`
	Latitude         float64   `json:"Latitude"`
	Longitude        float64   `json:"Longitude"`
	Owner            string    `json:"Owner"`
	Producer         string    `json:"Producer"`
	SmallCategory    string    `json:"SmallCategory"`
	Status           string    `json:"Status"`
	MyBidStatus		 string    `json:"My Bid Status"`
	Error string `json:"Error"`
}

const (
	earthRadius = 6378137.0
	// requestedTokenNum int = 10
	// batteryLife = 10 //%
	pricePerMater = 0.000001
	kmPerBattery = 0.05 // battery(%) * kmPerBattery = x km
	// myLatitude = 35.5552824466371 //0から89まで
	// myLongitude = 139.65527497388206
	// username = "user2"
	layout = "2006-01-02T15:04:05+09:00"
)

func Buy(contract *client.Contract, input Input) ([]Energy, error) {
	// batteryLifeから検索範囲決定
	searchRange := (100 - float64(input.BatteryLife)) * kmPerBattery * 1000 // 1000m->500mに変更
	fmt.Printf("searchRange:%g\n", searchRange)

	var tokenNum int = input.Token
	// var errEnergies []Energy

	lowerLat, upperLat, lowerLng, upperLng := determineRange(searchRange, input.Latitude, input.Longitude)
	energies, err := queryByLocationRange(contract, lowerLat, upperLat, lowerLng, upperLng)
	if err != nil {
		fmt.Println("query error")
		return energies, err
	}
	if(len(energies) == 0){
		return energies, nil
	}
	
	// fmt.Println(energies)
	fmt.Printf("length of energies: %d\n", len(energies))

	timestamp := time.Now()
	auctionStartTimeCompare := timestamp.Add(time.Minute * -5)

	validEnergies := []Energy{}

	for _, energy := range energies {
		distance := distance(input.Latitude, input.Longitude, energy.Latitude, energy.Longitude)
		if energy.Owner != input.User && distance <= searchRange && auctionStartTimeCompare.After(energy.AuctionStartTime) == false {
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

		tempSuccess := bid(contract, validEnergies, bidNum, input)

		success = append(success, tempSuccess...)
		validEnergies = validEnergies[bidNum:]
		tokenNum -= len(tempSuccess)
	}

	return success, nil
	
}

func BidResult(contract *client.Contract, successEnergy []Energy, input Input) { 

	// input:success List, input
	// const length = len(success)
	// var result[length] int
	var success []BidResultEnergy
	for i := 0; i < len(successEnergy); i++{
		token := BidResultEnergy{DocType: successEnergy[i].DocType, UnitPrice: successEnergy[i].UnitPrice, BidPrice: successEnergy[i].BidPrice, 
			GeneratedTime: successEnergy[i].GeneratedTime, AuctionStartTime: successEnergy[i].AuctionStartTime, ID: successEnergy[i].ID, 
			LargeCategory: successEnergy[i].LargeCategory, Latitude: successEnergy[i].Latitude, Longitude: successEnergy[i].Longitude, 
			Owner: successEnergy[i].Owner, Producer: successEnergy[i].Producer, SmallCategory: successEnergy[i].SmallCategory, Error: successEnergy[i].Error, 
			Status:successEnergy[i].Status}
		success = append(success, token)
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
			auctionEndToken, err := readToken(contract, success[i].ID)
			if err != nil {
				// できたらHTTP
			} else {
				auctionEndToken.Error = "OK"
			}
			if (auctionEndToken.Owner == input.User) {
				fmt.Println("you are a winner.")
				success[i].MyBidStatus = "win"
			} else {
				fmt.Println("you are a loser.")
				success[i].MyBidStatus = "lose"
			}
		}(i)
	}
	fmt.Println("BidResult:")
	fmt.Println(success)
	wg.Wait()
}

// 現在不使用
func HttpPostBidToken(energies []Energy) {
	const URL = "https://webhook.site/ba5e750f-7ffd-437b-962b-02ea67be8ca6"

	/*for _, energy := range energies{
		httpPost(energy);
	}*/

	energiesJson, err := json.Marshal(energies)
	if err != nil {
		fmt.Println(err)
	}
	res, err2 := http.Post(URL, "application/json", bytes.NewBuffer(energiesJson))
	defer res.Body.Close()

	if err2 != nil {
		fmt.Println(err2)
	} else {
		fmt.Println(res.Status)
	}
}

func httpPost(energy Energy, input Input) {
	const URL = "http://localhost:8090/bid"

	type BidToken struct {
		CarId string `json:"CarId"`
		CarEnergy int `json:"CarEnergy"`
		CarRadius float64 `json:"CarRadius"`
		CarLat float64 `json:"CarLat"`
		CarLon float64 `json:"CarLon"`
		Price float64 `json:"Price"`
		TokenId string `json:"TokenId"`
	}

	var token BidToken
	token.CarId = input.User
	token.CarEnergy = input.BatteryLife
	token.CarRadius = (100 - float64(input.BatteryLife)) * kmPerBattery
	token.CarLat = input.Latitude
	token.CarLon = input.Longitude
	price := energy.BidPrice
	token.Price = (math.Round(price * 100000)) / 100000
	fmt.Println(token.Price);
	// token.Price = energy.BidPrice
	token.TokenId = energy.ID

	fmt.Println(token)

	tokenJson, err := json.Marshal(token)
	if err != nil {
		fmt.Println(err)
	} else {
		res, err2 := http.Post(URL, "application/json", bytes.NewBuffer(tokenJson))
		if err2 != nil {
			fmt.Println(err2)
		} else {
			defer res.Body.Close()
			fmt.Println(res.Status)
		}
	}
}

func readToken(contract *client.Contract, energyId string) (Energy, error) {
	fmt.Printf("Async Submit Transaction: ReadToken'\n")
	var result Energy
	evaluateResult, err := contract.EvaluateTransaction("ReadToken", energyId)
	if err != nil {
		return result, err
		// panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	//result := formatJSON(evaluateResult)

	// result := Energy{}

	err = json.Unmarshal(evaluateResult, &result)
	if (err != nil) {
		return result, err
		// fmt.Printf("unmarshal error")
	}
	result.Error = "OK"
	return result, nil

	//fmt.Printf("*** Result:%s\n", result)
}

func bid(contract *client.Contract, energies []Energy, bidNum int, input Input) []Energy {
	successEnergy := []Energy{}
	//leftEnergy := energies
	
	c := make(chan Energy)

	for i := 0; i < bidNum; i++ {

		go func(i int, c chan Energy){
			fmt.Printf("id:%s, auctionStartTime:%s\n", energies[i].ID, energies[i].AuctionStartTime.Format(layout))
			message, err := bidOnToken(contract, energies[i].ID, energies[i].BidPrice, input.User)
			if err != nil {
				energies[i].Error = "bidOnTokenError: " + err.Error()
				c <- energies[i]
			}
			fmt.Println(message)
			if (message == "your bid was successful") {
				go httpPost(energies[i], input)
				bidResult, err := readToken(contract, energies[i].ID)
				if err != nil {
					energies[i].Error = "readTokenError: " + err.Error()
					// energies[i].MyBidStatus = err.Error()
					c <- energies[i]
				} else{
					bidResult.Error = "OK"
				}
				c <- bidResult
				// successEnergy = append(successEnergy, bidResult)
			// auctionstart + 5min 経ったら見に行く
			} else {
				energies[i].Error = "OK"
				c <- energies[i]
			}
		}(i, c)

	}

	for i := 0; i < bidNum; i++ {
		energy := <-c
		if (energy.Owner == input.User && energy.Error == "OK") {
			successEnergy = append(successEnergy, energy)
		}
	}

	return successEnergy
}

func bidOnToken(contract *client.Contract, energyId string, bidPrice float64, username string) (string, error) {
	//fmt.Printf("Evaluate Transaction: BidOnToken, function returns asset attributes\n")
	var timestamp = time.Now()
	var stringTimestamp = timestamp.Format(layout)
	var stringBidPrice = strconv.FormatFloat(bidPrice, 'f', -1, 64)
	//fmt.Printf("id:%s, timestamp:%s, price:%s\n", energyId, stringTimestamp, stringBidPrice)
	evaluateResult, err := contract.SubmitTransaction("BidOnToken", energyId, username, stringBidPrice, stringTimestamp)
	if err != nil {
		return "", err
		// panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	//result := formatJSON(evaluateResult)
	message := string(evaluateResult)
	/* "your bid was successful" */
	return message, nil
}


func determineRange(length float64, myLatitude float64, myLongitude float64) (lowerLat float64, upperLat float64, lowerLng float64, upperLng float64) {
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

func queryByLocationRange(contract *client.Contract, lowerLat float64, upperLat float64, lowerLng float64, upperLng float64) ([]Energy, error) {
	strLowerLat := strconv.FormatFloat(lowerLat, 'f', -1, 64)
	strUpperLat := strconv.FormatFloat(upperLat, 'f', -1, 64)
	strLowerLng := strconv.FormatFloat(lowerLng, 'f', -1, 64)
	strUpperLng := strconv.FormatFloat(upperLng, 'f', -1, 64)

	fmt.Printf("Async Submit Transaction: QueryByLocationRange'\n")

	result := []Energy{}
	evaluateResult, err := contract.EvaluateTransaction("QueryByLocationRange", "generated", strLowerLat, strUpperLat, strLowerLng, strUpperLng)
	if err != nil {
		return result, err
		// panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}

	fmt.Println(len(evaluateResult))

	err = json.Unmarshal(evaluateResult, &result)
	if(err != nil && len(evaluateResult) > 0) {
		return result, err
	}

	return result, nil

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
