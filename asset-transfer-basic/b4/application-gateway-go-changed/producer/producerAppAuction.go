package producer

import (
	"fmt"
	"time"
	"log"
	"encoding/json"
	"sort"
	//"math/rand"
	//"sync"
	
	"github.com/hyperledger/fabric-gateway/pkg/client"
)

func Auction(contract *client.Contract, energy Energy) {
	timer := time.NewTimer(time.Duration(Interval * 60 * 1000000000 - ((time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime - energy.GeneratedTime)) * time.Nanosecond / time.Duration(Speed))
	endTimer := time.NewTimer(time.Duration((EndTime - ((time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime)) / Speed) * time.Nanosecond)

	auctionEndCount := 0
	select {
	case <- timer.C:
		ticker := time.NewTicker(time.Duration(Interval * 60 * 1000000000) * time.Nanosecond / time.Duration(Speed))

		timestamp := (time.Now().UnixNano() -Diff - StartTime) * Speed + StartTime

		// fmt.Printf("auctionEndCall: id: %s, count: %d\n", energy.ID, auctionEndCount)
		isSold, err := auctionEnd(contract, energy, timestamp)
		if err != nil {
			log.Println(err)
			endTimer.Stop()
			ticker.Stop()
			panic(err)
		}
		if isSold {
			return
		}

		for i := 0; i < int(TokenLife / Interval); i++ {
			auctionEndCount++
			select {
			case <- ticker.C:
				timestamp := (time.Now().UnixNano() -Diff - StartTime) * Speed + StartTime
				// log.Printf("auctionEndCall: id: %s, count: %d, timestamp: %d\n", energy.ID, auctionEndCount, timestamp)
				
				isSold, err := auctionEnd(contract, energy, timestamp)
				if err != nil {
					fmt.Println(err)
					endTimer.Stop()
					ticker.Stop()
					panic(err)
				}
				
				if isSold {
					ticker.Stop()
					return
				}
			case <- endTimer.C:
				ticker.Stop()
				return
			}
		}
	case <-  endTimer.C:
		timer.Stop()
		return
	}

}

func auctionEnd(contract *client.Contract, energy Energy, timestamp int64) (bool, error) {
	isSold := false
	var message string
	var err error

	var bidInput []EndInput
	var soldAmount float64 = 0
	bidList, err := auctionEndQuery(contract, energy.ID, timestamp)
	if err != nil {
		if err.Error() == "time up" {
			// timeup
			return true, nil
		} else {
			return false, err
		}
	}
	if (timestamp <= energy.GeneratedTime + TokenLife * 60 * 1000000000 && len(bidList) == 0) {
		// log.Printf("%s auction end : no bidList, %v, now:%v\n", energy.ID, timestamp, ((time.Now().Unix() -Diff - StartTime) * Speed + StartTime))
		return isSold, nil
	}
	// log.Printf("%s length : %d\n", energy.ID, len(bidList))
	for i := 0; i < len(bidList); i++ {
		if (bidList[i].BidAmount < energy.Amount) {
			input := EndInput{ID: bidList[i].ID, Amount:bidList[i].BidAmount}
			energy.Amount -= bidList[i].BidAmount
			soldAmount += bidList[i].BidAmount
			bidInput = append(bidInput, input)
		} else {
			input := EndInput{ID: bidList[i].ID, Amount: energy.Amount}
			soldAmount += energy.Amount
			energy.Amount = 0
			bidInput = append(bidInput, input)
			isSold = true
			break
		}
	}
	energyInput := EndInput{ID: energy.ID, Amount: soldAmount, Time: timestamp}
	// log.Printf("%s soldAmount: %v\n", energy.ID, soldAmount)
	if (len(bidInput) == 0) {
		bidInput = append(bidInput, EndInput{ID: "old", Amount: 0})
	}
	message, err = auctionEndTransaction(contract, energyInput, bidInput)
	if err != nil {
		// timeup
		if err.Error() == "time up" {
			return true, nil
		} else {
			return false, err
		}
	}
// 	log.Printf("producer auction end: %s, %s, %s %vWh\n", energy.Producer, energy.ID, message, energyInput.Amount)
	if (message == "the energy was generated more than 30min ago. This was not sold." || message == "auction end") {
		if(message == "the energy was generated more than 30min ago. This was not sold.") {
			//fmt.Printf("ID:%v, %v\n", energy.ID, message)
		}
		isSold = true
	}

	return isSold, nil
}

func auctionEndQuery(contract *client.Contract, energyId string, timestamp int64) ([]Energy, error) {
	var bidList []Energy
	sTimestamp := fmt.Sprintf("%v", timestamp)
	loopCount := 0
	queryLoop:
	for {
		if ((time.Now().UnixNano() -Diff - StartTime) * Speed + StartTime > EndTime) {
			return bidList, fmt.Errorf("time up")
		}
		evaluateResult, err := contract.EvaluateTransaction("AuctionEndQuery", energyId, sTimestamp)
		if err != nil {
			// log.Printf("auction end query error: %v\n", err.Error())
			loopCount++
			if loopCount > 5 {
				log.Printf("auction end query error: %v, %v\n", energyId, err)
				//timestamp := time.Now().UnixNano()
				//r := rand.New(rand.NewSource(timestamp))
				//timer := time.NewTimer(time.Duration(r.Intn(3)) * time.Second + time.Duration(r.Intn(1000)) * time.Millisecond)
				//<- timer.C
				// return bidList, err
				//panic(err)
			}
		} else {
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
		
			/*sort.SliceStable(bidList, func(i, j int) bool {
				return bidList[i].Priority > bidList[j].Priority
			})
		
			sort.SliceStable(bidList, func(i, j int) bool {
				return bidList[i].BidPrice > bidList[j].BidPrice
			})*/
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
	loopCount := 0
	loop:
	for {
		if ((time.Now().UnixNano() -Diff - StartTime) * Speed + StartTime > EndTime) {
			return "", fmt.Errorf("time up")
		}
		submitResult, err := contract.SubmitTransaction("AuctionEnd", string(energyJSON), string(bidJSON))
		if err != nil {
			loopCount++
			if (loopCount > 5) {
				fmt.Printf("producer auction end error:%v\n", err)
				// timestamp := time.Now().UnixNano()
				//r := rand.New(rand.NewSource(timestamp))
				//timer := time.NewTimer(time.Duration(r.Intn(3)) * time.Second + time.Duration(r.Intn(1000)) * time.Millisecond)
				//<- timer.C
				panic(err)
				//return "", err
			}
			// log.Printf("producer auction end error: %v\n", err.Error())
			if (err.Error() == "energy amount is wrong" || err.Error() == "the energy is alive" || err.Error() == "energy ID is wrong" || err.Error() == "bid amount is wrong") {
				return message, nil
			}
		} else { 
			message = string(submitResult)
			break loop
		}
	}
	return message, nil
}

/*
func auctionEndTransaction(contract *client.Contract, energyId string, timestamp string) (string, error) {
	// fmt.Printf("Evaluate Transaction: auctionEnd\n")
	
	evaluateResult, err := contract.SubmitTransaction("AuctionEnd", energyId, timestamp)
	if err != nil {
		return "", err
	}
	fmt.Printf("evaluateResult length : %v\n", len(evaluateResult))
	message := string(evaluateResult)

	// fmt.Printf("*** %s Result:%s\n", energyId, massage)
	return message, nil
}*/
