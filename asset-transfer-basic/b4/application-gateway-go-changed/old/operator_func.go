/*
Copyright 2021 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
// 需要家
// Org2のユーザで実行

package main

import (
	//"bytes"
	//"context"
	"encoding/json"
	//"errors"
	"fmt"
	"time"
	"strconv"
	"errors"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	//"github.com/hyperledger/fabric-protos-go-apiv2/gateway"
	//"google.golang.org/grpc/status"
)

// var assetId = fmt.Sprintf("energy%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6)

type Solor struct {
	Month  int
	Hour int
	Price float64
}

const (
	totalDataNumber = 12
	hoursAdayHas = 24
)

func UpdateSolorUnitPrice(contract *client.Contract) {

	priceList := price()

	nowTime := time.Now()
	var err error
	err = errors.New("default error")
	for err != nil {
		err = updateSolor(contract, priceList)
	}

	next := time.Date(nowTime.Year(), nowTime.Month(), nowTime.Day(), nowTime.Hour() + 1, 0, 0, 0, time.Local)
	fmt.Println(next.Sub(nowTime))
	timer := time.NewTimer(next.Sub(nowTime))
	err = errors.New("default error")
	<-timer.C
	for err != nil {
		err = updateSolor(contract, priceList)
	}
	
	ticker := time.NewTicker(time.Hour * 1)
	for {
		err = errors.New("default error")
		<-ticker.C
		//cerr := updateSolor(contract, priceList)
		for err != nil {
			err = updateSolor(contract, priceList)
		}
	}

}

func updateSolor(contract *client.Contract, priceList [totalDataNumber][hoursAdayHas]float64) error {
	nowTime := time.Now()
	month := int(nowTime.Month())
	hour := int(nowTime.Hour())
		
	//month: 1-12, hour:0-23
	price := priceList[month - 1][hour]
	fmt.Printf("month:%d, hour:%d, price:%g\n", month, hour, price)
	err := update(contract, "solar", price)
	if err != nil {
		return err
	}
	return nil
}

func update(contract *client.Contract, smallCategory string, unitPrice float64) error {

	fmt.Printf("Submit Transaction: changeUnitPrice\n")
	var timestamp = time.Now()
	var layout = "2006-01-02T15:04:05+09:00"
	var stringTimestamp = timestamp.Format(layout)
	var stringUnitPrice = strconv.FormatFloat(unitPrice, 'f', -1, 64)
	fmt.Println(smallCategory)
	fmt.Println(stringUnitPrice)
	fmt.Println(stringTimestamp)


	// smallCategory string, newUnitPrice float64, timestamp time.Time
	_, err := contract.SubmitTransaction("UpdateUnitPrice", smallCategory, stringUnitPrice, stringTimestamp)
	if err != nil {
		return err
		// panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")

	energy, err := readToken(contract, "solar-power-cost")
	if err != nil {
		return err
	}
	fmt.Println(energy)
	return nil

}

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
	//MyBidStatus		 string    `json:"My Bid Status"`
}

func readToken(contract *client.Contract, energyId string) (Energy, error) {
	fmt.Printf("Async Submit Transaction: ReadToken'\n")
	result := Energy{}
	evaluateResult, err := contract.EvaluateTransaction("ReadToken", energyId)
	if err != nil {
		return result, err
		// panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	//result := formatJSON(evaluateResult)

	err = json.Unmarshal(evaluateResult, &result)
	if (err != nil) {
		return result, err
		// fmt.Printf("unmarshal error")
	}

	return result, nil

	//fmt.Printf("*** Result:%s\n", result)
}

func price() [totalDataNumber][hoursAdayHas]float64 {
	maxPrice := 0.025
	minPrice := 0.015

	temperatureData := [totalDataNumber][hoursAdayHas]float64{
		{51, 50, 49, 45, 48, 52, 57, 74, 96, 120, 135, 137, 143, 146, 139, 129, 115, 106, 99, 88, 90, 87, 84, 79}, 
		{0, 0, 0, 0, 0, 0, 0, 1, 5, 6, 9, 10, 10, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 
		{69, 63, 52, 47, 43, 37, 39, 49, 63, 78, 79, 91, 93, 100, 106, 110, 103, 97, 82, 77, 72, 72, 71, 69},
		{72, 71, 67, 54, 53, 55, 61, 76, 83, 95, 116, 132, 135, 138, 141, 135, 134, 130, 127, 102, 92, 86, 79, 75}, 
		{162, 153, 153, 141, 132, 136, 147, 167, 193, 184, 201, 224, 246, 254, 247, 246, 238, 230, 206, 203, 191, 185, 175, 169}, 
		{218, 216, 210, 205, 205, 213, 217, 225, 232, 241, 253, 259, 260, 267, 267, 250, 249, 235, 227, 221, 218, 215, 212, 209}, 
		{216, 214, 215, 222, 223, 217, 223, 242, 238, 253, 247, 271, 275, 289, 290, 280, 264, 257, 248, 242, 237, 235, 232, 228}, 
		{246, 243, 237, 229, 227, 225, 227, 233, 250, 260, 249, 267, 267, 270, 274, 266, 268, 260, 249, 240, 233, 230, 226, 221},
		{234, 229, 227, 222, 212, 207, 213, 223, 236, 241, 247, 259, 264, 242, 234, 220, 219, 215, 218, 217, 199, 185, 182, 181},
		{212, 217, 217, 219, 219, 221, 223, 209, 200, 197, 195, 196, 193, 193, 194, 197, 196, 197, 197, 192, 196, 193, 197, 196},
		{151, 145, 137, 130, 126, 121, 118, 127, 132, 163, 167, 173, 191, 196, 189, 176, 164, 156, 143, 147, 149, 142, 138, 131},
		{116, 110, 102, 88, 82, 80, 79, 81, 80, 82, 79, 80, 76, 75, 79, 81, 78, 80, 79, 77, 71, 65, 53, 48}}

	solarRadiationData := [totalDataNumber][hoursAdayHas]float64{
		{0, 0, 0, 0, 0, 0, 0, 72, 166, 249, 295, 320, 312, 276, 206, 111, 10, 0, 0, 0, 0, 0, 0, 0}, 
		{0, 0, 0, 0, 0, 0, 2, 24, 105, 175, 266, 319, 310, 154, 62, 31, 9, 0, 0, 0, 0, 0, 0, 0}, 
		{0, 0, 0, 0, 0, 0, 16, 103, 204, 278, 320, 352, 298, 294, 201, 131, 37, 9, 0, 0, 0, 0, 0, 0}, 
		{0, 0, 0, 0, 0, 3, 54, 89,170, 305, 354, 380, 358, 329, 270, 172, 60, 15, 0, 0, 0, 0, 0, 0}, 
		{0, 0, 0, 0, 1, 15, 74, 155, 227, 234, 315, 354, 349, 313, 252, 133, 48, 20, 3, 0, 0, 0, 0, 0}, 
		{0, 0, 0, 0, 4, 21, 82, 150, 233, 293, 334, 350, 339, 308, 236, 179, 94, 35, 7, 0, 0, 0, 0, 0}, 
		{0, 0, 0, 0, 3, 14, 29, 107, 134, 260, 121, 311, 323, 290, 249, 173, 63, 39, 8, 0, 0, 0, 0, 0}, 
		{0, 0, 0, 0, 1, 7, 25, 46, 163, 232, 142, 245, 199, 160, 237, 175, 94, 37, 5, 0, 0, 0, 0, 0}, 
		{0, 0, 0, 0, 0, 7, 23, 78, 160, 109, 134, 317, 298, 70, 46, 31, 23, 7, 1, 0, 0, 0, 0, 0}, 
		{0, 0, 0, 0, 0, 3, 7, 10, 18, 18, 32, 35, 40, 35, 27, 22, 7, 3, 0, 0, 0, 0, 0, 0}, 
		{0, 0, 0, 0, 0, 0, 31, 37, 53, 143, 142, 155, 261, 269, 199, 110, 31, 0, 0, 0, 0, 0, 0, 0}, 
		{0, 0, 0, 0, 0, 0, 3, 7, 18, 19, 29, 30, 29, 21, 21, 9, 3, 0, 0, 0, 0, 0, 0, 0}}

	var output [12][24]float64
	var priceList [12][24]float64
	var maxOutput float64
	var minOutput float64


	annualIrradiationDeviationFactor := 0.97 // 日射量年変動補正係数
	efficiencyDeviationFactor := 0.95 // 経時変化補正係数
	arrayLoadMatchingCorrectionFactor := 0.94 // アレイ負荷整合補正係数
	arrayLoadCorrectionFactor := 0.97 // アレイ回路補正整合補正係数
	inerterEffectiveEnergyEfficiency := 0.90 // インバータ実効効率

	temperatureFactor := -0.45

	maxOutput = 0
	minOutput = 100000
	for i := 0; i < totalDataNumber; i++ {
		for j := 0; j < hoursAdayHas; j++ {
			basicDesignFactor := annualIrradiationDeviationFactor * efficiencyDeviationFactor * 
			arrayLoadMatchingCorrectionFactor * arrayLoadCorrectionFactor * inerterEffectiveEnergyEfficiency
			
			totalDesignFactor := basicDesignFactor * (1 + temperatureFactor * (temperatureData[i][j] * 0.1 - 25) / 100)

			output[i][j] = totalDesignFactor * solarRadiationData[i][j] * 10 / 3.6
			if output[i][j] > maxOutput {
				maxOutput = output[i][j]
			}
			if output[i][j] < minOutput {
				minOutput = output[i][j]
			}

		}
	}

	outPutDifferenceMaxMin := maxOutput - minOutput
	priceDifference := maxPrice - minPrice


	for i := 0; i < totalDataNumber; i++ {
		for j := 0; j < hoursAdayHas; j++ {
			outputDifferenceFromMin := output[i][j] - minOutput
			priceList[i][j] = maxPrice - priceDifference * (outputDifferenceFromMin / outPutDifferenceMaxMin)
		}
	}

	return priceList
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
