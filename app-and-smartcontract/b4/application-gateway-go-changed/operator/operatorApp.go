package operator

import (
	"fmt"
	"time"
	"encoding/json"
	
	"github.com/hyperledger/fabric-gateway/pkg/client"
)


func Init(contract *client.Contract) {
	fmt.Printf("Submit Transaction: InitLedger, function creates the initial set of assets on the ledger \n")

	_, err := contract.SubmitTransaction("InitLedger")
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

func Operate(contract *client.Contract, output [dayNum][hourNum]float64, cat string) {
	var priceList [dayNum][hourNum] float64

	endTimer := time.NewTimer(time.Duration((EndTime - ((time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime)) / Speed) * time.Nanosecond)

	maxPrice := 0.025
	minPrice := 0.015

	var maxOutput float64 = 0
	var minOutput float64 = 100000
	for i := 0; i < dayNum; i++ {
		for j := 0; j < hourNum; j++ {
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

	for i := 0; i < dayNum; i++ {
		for j := 0; j < hourNum; j++ {
			outputDifferenceFromMin := output[i][j] - minOutput
			priceList[i][j] = maxPrice - priceDifference * (outputDifferenceFromMin / outPutDifferenceMaxMin)
		}
	}

	// fmt.Println(priceList)

	

	day := 0
	hour := StartHour
	timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
	_ = update(contract, priceList, cat, day, hour, timestamp)

	startNano := StartTime % 1000
	startMicro := (StartTime / 1000) % 1000
	startMilli := (StartTime / 1000000) % 1000
	startSecond := (StartTime / 1000000000) % 60
	startMinute := (StartTime / (60 * 1000000000)) % 60
	nextTime := StartTime - startMinute * 60 * 1000000000 - startSecond * 1000000000 - startMilli * 1000000 - startMicro * 1000 - startNano + 60 * 60 * 1000000000
	fmt.Printf("minute:%v, second:%v, milli:%v, miroco:%v, nano:%v, now: %v, next update price:%v\n", startMinute, startSecond, startMilli, startMicro, startNano, time.Unix(0, StartTime), time.Unix(0, nextTime))
	nextTimer := time.NewTimer(time.Duration(nextTime - ((time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime)) * time.Nanosecond / time.Duration(Speed))

	if (hour > 22) {
		day = 1
		hour = 0
	} else {
		hour++
	}
	
	select {
	case <- nextTimer.C:
		timestamp = (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
		fmt.Printf("UPDATE UNIT PRICE: %v\n", time.Unix(0, timestamp))
		_ =  update (contract, priceList, cat, day, hour, timestamp)
	case <- endTimer.C:
		timestamp = (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
		fmt.Printf("UPDATE UNIT PRICE END: %v\n", time.Unix(0, timestamp))
		nextTimer.Stop()
		return
	}

	ticker := time.NewTicker(1 * time.Hour / time.Duration(Speed))

	if (hour > 22) {
		day = 1
		hour = 0
	} else {
		hour++
	}

	for {
		select {
		case <- ticker.C:
			timestamp = (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
			fmt.Printf("UPDATE UNIT PRICE: %v\n", time.Unix(0, timestamp))
			_ = update (contract, priceList, cat, day, hour, timestamp)
			if (hour > 22) {
				day = 1
				hour = 0
			} else {
				hour++
			}
		case <- endTimer.C:
			timestamp = (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
			fmt.Printf("UPDATE UNIT PRICE END: %v\n", time.Unix(0, timestamp))
			nextTimer.Stop()
			return
		}
	}
}
func update (contract *client.Contract, priceList [dayNum][hourNum]float64, cat string, day int, hour int, timestamp int64) error {
	// fmt.Printf("Submit Transaction: changeUnitPrice\n")
	sTimestamp := fmt.Sprintf("%d", timestamp)
	sPrice := fmt.Sprintf("%v", priceList[day][hour])

	// fmt.Printf("category:%s, price:%s, timestamp:%s\n", cat, sPrice, sTimestamp)

	// smallCategory string, newUnitPrice float64, timestamp time.Time
	for {
		_, err := contract.SubmitTransaction("UpdateUnitPrice", cat, sPrice, sTimestamp)
		if (err != nil) {
			fmt.Printf("update unit error: %v\n", err)
		} else {
			break
		}
	}
	// fmt.Printf("*** Transaction committed successfully\n")

	/*readName := fmt.Sprintf("%s-power-cost", cat)
	energy, err := readToken(contract, readName)
	if err != nil {
		return err
	}
	fmt.Println(energy)*/
	return nil
}


func readToken(contract *client.Contract, energyId string) (Energy, error) {
	fmt.Printf("Async Submit Transaction: ReadToken'\n")
	result := Energy{}
	for {
		evaluateResult, err := contract.EvaluateTransaction("ReadToken", energyId)
		if err != nil {
			fmt.Println(err)
		}
		err = json.Unmarshal(evaluateResult, &result)
		if (err != nil) {
			return result, err
		} else {
			break
		}
	}

	return result, nil
}