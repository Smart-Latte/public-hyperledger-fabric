/*
Copyright 2021 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
// 需要家
// Org2のユーザで実行

package main

import (
	"encoding/json"
	"fmt"
	"time"
	"math/rand"
	"strconv"
	"net/http"
	"bytes"

	"github.com/hyperledger/fabric-gateway/pkg/client"
)

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
	Error string `json:"Error"`
}

const (
	earthRadius = 6378137.0
	auctionEndMax      = 6
	auctionEndInterval = 5
	layout = "2006-01-02T15:04:05+09:00"
)

func Create(contract *client.Contract, input Input) (Energy, time.Time) {
	var largeCategory string
	if (input.Category == "solar" || input.Category == "wind") {
		largeCategory = "green"
	} else {
		largeCategory = "depletable"
	}
	var timestamp = time.Now()
	rand.Seed(time.Now().UnixNano())
	id := timestamp.Format(layout) + input.User + "-" + strconv.Itoa(rand.Intn(10000))

	energy, err := createToken(contract, id, timestamp, largeCategory, input.Category, input)
	if err != nil {
		energy.Error = "createToken: " + err.Error()
	}
	return energy, timestamp
}

func Auction(contract *client.Contract, energy Energy, timestamp time.Time, input Input) {
	ticker := time.NewTicker(time.Minute * auctionEndInterval)
	count := 0
loop:
	for {
		select {
		case <-ticker.C:
			count++
			fmt.Printf("count:%d\n", count)
			auctionEndTimestamp := timestamp.Add(time.Minute * time.Duration(count*auctionEndInterval))
			massage, err := auctionEnd(contract, energy.ID, auctionEndTimestamp, input)
			if (err != nil) {
				fmt.Println(err)
			}
			fmt.Printf("timestamp:%v, id:%s\n", auctionEndTimestamp, energy.ID)

			stopmassage1 := "the energy " + energy.ID + " was generated more than 30min ago. This was not sold."
			stopmassage2 := "the energy " + energy.ID + " was sold. It was generetad more than 30min ago."
			stopmassage3 := "the energy " + energy.ID + " was sold"
			if massage == stopmassage1 || massage == stopmassage2 || massage == stopmassage3 {
				ticker.Stop()
				break loop
			} else if count == auctionEndMax - 1 {
				err = discountUnitPrice(contract, energy.ID)
				fmt.Println("discount")
			} else if count == auctionEndMax {
				ticker.Stop()
				fmt.Println("final")
				break loop
			} 
		}
	}
	resultEnergy, err := readToken(contract, energy.ID)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(resultEnergy)
		httpPostAuctionEnd(resultEnergy)
	}
}

func createToken(contract *client.Contract, energyId string, timestamp time.Time, largeCAT string, smallCAT string, input Input) (Energy, error) {
	fmt.Printf("Submit Transaction: CreateToken, creates new token with ID, Latitude, Longitude, Owner, Large Category, Small Category and timestamp \n")
	var stringTimestamp = timestamp.Format(layout)
	var stringLatitude = strconv.FormatFloat(input.Latitude, 'f', -1, 64)
	var stringLongitude = strconv.FormatFloat(input.Longitude, 'f', -1, 64)
	var energy Energy
	_, err := contract.SubmitTransaction("CreateToken", energyId, stringLatitude, stringLongitude, input.User, largeCAT, smallCAT, stringTimestamp)
	if err != nil {
		return energy, err
	}

	fmt.Printf("*** Transaction committed successfully\n")

	energy, err = readToken(contract, energyId)
	if err != nil {
		return energy, err
	}

	return energy, nil
}

func discountUnitPrice(contract *client.Contract, energyId string) error {

	_, err := contract.SubmitTransaction("DiscountUnitPrice", energyId)
	if err != nil {
		return err
	}
	return nil
}

func auctionEnd(contract *client.Contract, energyId string, timestamp time.Time, input Input) (string, error) {
	fmt.Printf("Evaluate Transaction: auctionEnd\n")
	var stringTimestamp = timestamp.Format(layout)
	fmt.Println(energyId)
	fmt.Println(input.User)
	fmt.Println(stringTimestamp)
	evaluateResult, err := contract.SubmitTransaction("AuctionEnd", energyId, input.User, stringTimestamp)
	if err != nil {
		return "", err
	}
	massage := string(evaluateResult)

	fmt.Printf("*** Result:%s\n", massage)
	return massage, nil
}

func readToken(contract *client.Contract, energyId string) (Energy, error) {
	fmt.Printf("Async Submit Transaction: ReadToken\n")
	var energy Energy
	evaluateResult, err := contract.EvaluateTransaction("ReadToken", energyId)
	if err != nil {
		return energy, err
	}
	
	err = json.Unmarshal(evaluateResult, &energy)
	if(err != nil) {
		return energy, err
	}

	return energy, nil
}

func HttpPostCreatedToken(energy Energy) {
	// const URL = "https://webhook.site/ba5e750f-7ffd-437b-962b-02ea67be8ca6"
	const URL = "http://localhost:8090/token"
	fmt.Println(energy)
	type CreateToken struct {
		TokenId          string    `json:"TokenId"`
		TokenPrice        float64   `json:"TokenPrice"`
		TokenLat         float64   `json:"TokenLat"`
		TokenLon        float64   `json:"TokenLon"`
	}

	var token CreateToken
	token.TokenId = energy.ID
	token.TokenLat = energy.Latitude
	token.TokenLon = energy.Longitude
	token.TokenPrice = energy.UnitPrice

	tokenJson, err := json.Marshal(token)
	if err != nil {
		fmt.Printf("err1")
		fmt.Println(err)
	}
	res, err2 := http.Post(URL, "application/json", bytes.NewBuffer(tokenJson))

	if err2 != nil {
		fmt.Printf("err2")
		fmt.Println(err2)
	} else {
		fmt.Printf("add data to map %v\n", res.Status)
		res.Body.Close()
	}
}

func httpPostAuctionEnd(energy Energy) {
	// const URL = "https://webhook.site/ba5e750f-7ffd-437b-962b-02ea67be8ca6"
	const URL = "http://localhost:8090/auction"

	type AuctionEndToken struct {
		WinnerCarId string `json:"WinnerCarId"`
		TokenId string `json:"TokenId"`
	}

	var token AuctionEndToken

	if energy.Owner != energy.Producer {
		token.WinnerCarId = energy.Owner
	} else {
		token.WinnerCarId = "-1"
	}

	token.TokenId = energy.ID
	

	tokenJson, err := json.Marshal(token)
	if err != nil {
		fmt.Println(err)
	}
	res, err2 := http.Post(URL, "application/json", bytes.NewBuffer(tokenJson))

	if err2 != nil {
		fmt.Println(err2)
	} else {
		fmt.Printf("auction end post %v\n", res.Status)
		res.Body.Close()
	}
}
