package consumer

import (
	//"bytes"
	"encoding/json"
	"fmt"
	"time"
	//"strconv"
	//"math"
	//"sort"
	"sync"
	//"net/http"
	
	"github.com/hyperledger/fabric-gateway/pkg/client"
)

/*const (
	earthRadius = 6378137.0
	pricePerMater = 0.000001
	kmPerBattery = 0.05 // battery(%) * kmPerBattery = x km
	layout = "2006-01-02T15:04:05+09:00"
)*/

func BidResult(contract *client.Contract, bidEnergies []Energy, data Data) (Data, error) {
	// endTimer := time.NewTimer(time.Duration((EndTime - ((time.Now().Unix() - Diff - StartTime) * Speed + StartTime)) / Speed) * time.Second)
	var wg sync.WaitGroup
	// endFlag := true

	for i := 0; i <  len(bidEnergies); i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			checkLoop:
			for {
				if ((time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime > EndTime) {
					return
				}
				bidToken, err := readToken(contract, bidEnergies[n].ID)
				if err != nil {
					return
				}
				// fmt.Printf("read bid token: ID: %s, energyID: %s, Status: %s, count: %d", bidToken.ID, bidToken.EnergyID, bidToken.Status, i)
				if (bidToken.Status == "success") {
					bidEnergies[n] = bidToken	
				}
				break checkLoop
			}
		}(i)
	}
	wg.Wait()

	if ((time.Now().UnixNano() -Diff - StartTime) * Speed + StartTime > EndTime) {
		return data, fmt.Errorf("time up")
	}
	data.GetAmount = 0
	data.GetSolar = 0
	data.GetWind = 0
	data.GetThermal = 0

	for i := 0; i < len(bidEnergies); i++ {
		if (bidEnergies[i].Status == "success") {

			data.GetAmount += bidEnergies[i].Amount
			switch bidEnergies[i].SmallCategory {
			case "solar":
				data.GetSolar += bidEnergies[i].Amount
			case "wind":
				data.GetWind += bidEnergies[i].Amount
			case "thermal":
				data.GetThermal += bidEnergies[i].Amount
			}
		}
	}
	return data, nil
}

func readToken(contract *client.Contract, id string) (Energy, error) {
	// endTimer := time.NewTimer(time.Duration((EndTime - ((time.Now().Unix() - Diff - StartTime) * Speed + StartTime)) / Speed) * time.Second)
	var energy Energy
	// fmt.Printf("Async Submit Transaction: ReadToken: %s\n", id)
	count := 0
	queryLoop:
	for {
		if ((time.Now().UnixNano() -Diff - StartTime) * Speed + StartTime > EndTime) {
			return energy, fmt.Errorf("time up")
		}
		evaluateResult, err := contract.EvaluateTransaction("readToken", id)
		if err != nil {
			fmt.Printf("BID RESULT ERROR: %v, %v\n", id, err.Error())
			count++
			return energy, err
				//panic(fmt.Errorf("bid result error %v, failed to evaluate transaction: %v\n", id, err))
		} else {
			// fmt.Printf("BID SUCCESS: %v\n", id)
			err = json.Unmarshal(evaluateResult, &energy)
			if(err != nil) {
				fmt.Printf("unmarshal error in queryInBidResult\n")
			} else {
				// fmt.Printf("%s break queryLoop\n", id)
				break queryLoop
			}
		} 	
	}
	return energy, nil
}
