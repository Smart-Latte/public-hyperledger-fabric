/*
Copyright 2021 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
// 需要家
// Org2のユーザで実行

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-protos-go-apiv2/gateway"
	"google.golang.org/grpc/status"
)

var assetId = fmt.Sprintf("energy%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6)


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
}

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
}

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

func ReadToken(contract *client.Contract) {
	fmt.Printf("Async Submit Transaction: ReadToken, updates existing asset owner'\n")

	evaluateResult, err := contract.EvaluateTransaction("ReadToken", assetId)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

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
