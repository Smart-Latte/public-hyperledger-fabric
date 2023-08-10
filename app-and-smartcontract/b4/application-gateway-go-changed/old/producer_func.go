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
	"math/rand"
	"strconv"
	"net/http"
	"bytes"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	//"github.com/hyperledger/fabric-protos-go-apiv2/gateway"
	//"google.golang.org/grpc/status"
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
	//myLatitude         = "35.54738979492469" //0-89
	//myLongitude        = "139.67098316696772"
	//username           = "User1"
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
	// create id
	id := timestamp.Format(layout) + input.User + "-" + strconv.Itoa(rand.Intn(10000))

	//var energy Energy
	// create token
	energy, err := createToken(contract, id, timestamp, largeCategory, input.Category, input)
	if err != nil {
		energy.Error = "createToken: " + err.Error()
	}
	return energy, timestamp
	// go auction()
	// Notification of errors?
	// fmt.Println(energy)

}

func Auction(contract *client.Contract, energy Energy, timestamp time.Time, input Input) {
	ticker := time.NewTicker(time.Minute * auctionEndInterval)
	count := 0
	// Check for bidders every 5 minutes
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
				// discount the auction between 25min and 30min
				err = discountUnitPrice(contract, energy.ID)
				fmt.Println("discount")
			} else if count == auctionEndMax {
				// last auction
				ticker.Stop()
				fmt.Println("final")
				break loop
			} 
		}
	}
	// http post
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
	defer res.Body.Close()

	if err2 != nil {
		fmt.Printf("err2")
		fmt.Println(err2)
	} else {
		fmt.Println(res.Status)
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
	defer res.Body.Close()

	if err2 != nil {
		fmt.Println(err2)
	} else {
		fmt.Println(res.Status)
	}
}

/*
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
}*/

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
/*
// Evaluate a transaction by assetID to query ledger state.
func BidOnToken(contract *client.Contract) {
	fmt.Printf("Evaluate Transaction: BidOnToken, function returns asset attributes\n")
	var timestamp = time.Now()
	var layout = "2006-01-02T15:04:00Z"
	var stringTimestamp = timestamp.Format(layout)
	evaluateResult, err := contract.SubmitTransaction("BidOnToken", assetId, "Mayuko", "1", stringTimestamp)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	//result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", evaluateResult)
}*/
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
}*/

// Submit transaction asynchronously, blocking until the transaction has been sent to the orderer, and allowing
// this thread to process the chaincode response (e.g. update a UI) without waiting for the commit notification
/*
func QueryByStatus(contract *client.Contract) {
	fmt.Printf("Async Submit Transaction: QueryByStatus'\n")

	evaluateResult, err := contract.EvaluateTransaction("QueryByStatus", "generated")
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}
*/
/*
func QueryByLocationRange(contract *client.Contract) {
	fmt.Printf("Async Submit Transaction: QueryByLocationRange'\n")

	evaluateResult, err := contract.EvaluateTransaction("QueryByLocationRange", "generated", "35.54", "35.55", "139.67", "139.68")
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}*/

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
/*func ExampleErrorHandling(contract *client.Contract) {
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
*/
/*
// Format JSON data
func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, " ", ""); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}
*/