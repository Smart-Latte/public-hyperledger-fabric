package producer

import (
	"fmt"
	"math/rand"
	"time"
	"encoding/json"

	"github.com/hyperledger/fabric-gateway/pkg/client"
)

func Create(contract *client.Contract, input Input) () {
	endTimer := time.NewTimer(time.Duration((EndTime - ((time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime)) / Speed) * time.Nanosecond)
	errCount := 0
	var energy Energy
	var err error

	createLoop:
	for {
		select {
		case <-endTimer.C:
			return
		default:
			energy, err = createToken(contract, input)
			if err != nil {
				if errCount > 3 {
					fmt.Printf("%s many create error: %v, %s\n", energy.ID, err, energy.Producer)
					return
				}
				//fmt.Printf("create Token Error: %s\n", err.Error())
				errCount++
				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				timer := time.NewTimer(time.Duration(r.Intn(1000000000)) * time.Nanosecond / time.Duration(Speed))
				<- timer.C
			} else {
				break createLoop
			}
		}
	}

	select {
	case <-endTimer.C:
		return
	default:
		// fmt.Printf("call auction %s\n", energy.ID)
		Auction(contract, energy)
	}
}

func createToken(contract *client.Contract, input Input) (Energy, error) {
	// fmt.Printf("Submit Transaction: CreateToken, creates new token with ID, Latitude, Longitude, Owner, Large Category, Small Category and timestamp \n")
	var largeCategory string
	if (input.Category == "solar" || input.Category == "wind") {
		largeCategory = "green"
	} else {
		largeCategory = "depletable"
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// create id
	id := fmt.Sprintf("%d%s-%d", input.Timestamp, input.User, r.Intn(10000))

	sTimestamp := fmt.Sprintf("%v", input.Timestamp)
	sLat := fmt.Sprintf("%v", input.Latitude)
	sLon := fmt.Sprintf("%v", input.Longitude)
	sAmo := fmt.Sprintf("%v", input.Amount)

	var energy Energy

	// fmt.Println("create")

	evaluateResult, err := contract.SubmitTransaction("CreateEnergyToken", id, sLat, sLon, input.User, sAmo, largeCategory, input.Category, sTimestamp)
	if err != nil {
		return energy, err
	}

	err = json.Unmarshal(evaluateResult, &energy)
	if err != nil {
		return energy, err
	}

	// fmt.Printf("*** %s: Transaction committed successfully\n", id)

	return energy, nil
}