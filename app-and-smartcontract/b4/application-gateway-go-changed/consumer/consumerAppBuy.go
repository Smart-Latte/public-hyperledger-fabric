package consumer

import (
	//"bytes"
	"encoding/json"
	"fmt"
	"time"
	"strconv"
	"math"
	"sort"
	//"sync"
	"math/rand"
	"log"
	"strings"
	//"net/http"
	
	"github.com/hyperledger/fabric-gateway/pkg/client"
)

type Output struct {
	Message string `json:"Message"`
	Amount float64 `json:"Amount"`
}

const (
	earthRadius = 6378137.0
	pricePerMater = 0.000001
	//kmPerBattery = 0.0325 // 0.065 // (100-battery(%)) * kmPerBattery = x km
	rangeMax = 6500
	rangeMin = 2600.0//3250.0
	layout = "2006-01-02T15:04:05+09:00"
)

func Bid(contract *client.Contract, data Data) ([]Energy, Data, error) {
	// endTimer := time.NewTimer(time.Duration((EndTime - ((time.Now().Unix() - Diff - StartTime) * Speed + StartTime)) / Speed) * time.Second)
	max := rangeMax * math.Sqrt(2)
	search := 1 - data.BatteryLife
	searchRange := search * (max - rangeMin) + rangeMin
	searchRange = 40000
	//searchRange := search * kmPerBattery * 1000 // 1000m->500mに変更
	/*if (searchRange < 3250) {
		searchRange = 3250
	} // 次は半分とか？20%現実的じゃない*/
	//searchRange += rangemin
	// searchRange = 200000
	//fmt.Printf("battery: %v, searchRange:%g\n", data.BatteryLife, searchRange)

	var energies []Energy
	var success []Energy
	var err error

	if ((time.Now().UnixNano() -Diff - StartTime) * Speed + StartTime > EndTime) {
		return success, data, fmt.Errorf("time up")
	}
	// var errEnergies []Energy

	lowerLat, upperLat, lowerLng, upperLng := determineRange(searchRange, data.Latitude, data.Longitude)

	energies, err = queryByLocationRange(contract, data.UserName, lowerLat, upperLat, lowerLng, upperLng)
	if err != nil {
		// time up
		return success, data, err
	}else if (len(energies) == 0){
		return success, data, nil
	}

	// fmt.Println(energies)
	// fmt.Printf("length of energies: %d\n", len(energies))
	// timestamp := (time.Now().Unix() - Diff - StartTime) * Speed + StartTime
	// auctionStartTimeCompare := timestamp - 60 * Interval

	validEnergies := []Energy{}
	generatedTimeCompare := (time.Now().UnixNano() -Diff - StartTime) * Speed + StartTime - (TokenLife - 1) * 60 * 1000000000

	for _, energy := range energies {
		distance := distance(data.Latitude, data.Longitude, energy.Latitude, energy.Longitude)
		if (distance <= searchRange && generatedTimeCompare < energy.GeneratedTime) {
			myBidPrice := energy.UnitPrice + distance * pricePerMater
			isOk, _ := bidOk(contract, energy.ID, myBidPrice, (1 - data.BatteryLife))
			if isOk {
				energy.Distance = distance
				energy.BidPrice = myBidPrice
				validEnergies = append(validEnergies, energy)
			}
			
			/*if (myBidPrice > energy.BidPrice || (myBidPrice == energy.BidPrice && (1 - data.BatteryLife) > energy.Priority)) {
				energy.BidPrice = myBidPrice
				validEnergies = append(validEnergies, energy)
			}*/
			// fmt.Println("it's valid")
			// fmt.Printf("id:%s, latitude:%g, longitude:%g, unitPrice:%g, distance:%g, bidPrice:%g\n", 
			// energy.ID, energy.Latitude, energy.Longitude, energy.UnitPrice, distance, energy.BidPrice)
		}else {
			// fmt.Println("it's invalid")
			// fmt.Printf("id:%s, latitude: %g, longitude:%g, unitPrice:%g, distance:%g, auctionStartTime:%d\n",
		// energy.ID, energy.Latitude, energy.Longitude, energy.UnitPrice, distance, energy.AuctionStartTime)
		}
	}

	r := rand.New(rand.NewSource((time.Now().UnixNano() -Diff - StartTime) * Speed + StartTime))
	r.Shuffle(len(validEnergies), func(i, j int) {
		validEnergies[i], validEnergies[j] = validEnergies[j], validEnergies[i]
	})
	/*sort.SliceStable(validEnergies, func(i, j int) bool {
		return validEnergies[i].GeneratedTime < validEnergies[j].GeneratedTime
	})*/
	sort.SliceStable(validEnergies, func(i, j int) bool {
		return validEnergies[i].BidPrice < validEnergies[j].BidPrice
	})
	/*
	if (data.BatteryLife < 0.5 && data.HighPrice == 1) {
		fmt.Printf("%v, %v, high price\n", data.UserName, data.BatteryLife)
		sort.SliceStable(validEnergies, func(i, j int) bool {
			return validEnergies[i].BidPrice > validEnergies[j].BidPrice
		})
	} else {
		//fmt.Printf("%v, %v, low price\n", data.UserName, data.BatteryLife)
		sort.SliceStable(validEnergies, func(i, j int) bool {
			return validEnergies[i].BidPrice < validEnergies[j].BidPrice
		})
	}*/
	
	//fmt.Println(validEnergies)

	/*fmt.Println("sort validEnergies")
	for i := 0; i < len(validEnergies) ; i++ {
		if (i < 7) {
			fmt.Printf("id: %s, bidPrice:%v, generatedTime:%v\n", validEnergies[i].ID, validEnergies[i].BidPrice, validEnergies[i].GeneratedTime)
		} else {break}
	}*/

	leftAmount := data.Requested
	
	loop:
		for {
			if ((time.Now().UnixNano() -Diff - StartTime) * Speed + StartTime > EndTime) {
				return success, data, fmt.Errorf("time up")
			}
			if(leftAmount == 0 || len(validEnergies) == 0) {
				break loop
			}
			// fmt.Printf("requested Amount:%g\n", leftAmount)
			// fmt.Printf("valid energy token:%d\n", len(validEnergies))
			want := leftAmount
			tokenCount := 0
			for i := 0; i < len(validEnergies); i++ {
				if (want == 0) {
					break
				}
				if (validEnergies[i].Amount > want) {
					validEnergies[i].Amount = want
					want = 0
					tokenCount++
					break
				} else {
					want -= validEnergies[i].Amount
					tokenCount++
				}
			}
			tempSuccess, tempAmount := bid(contract, validEnergies, tokenCount, data)

			success = append(success, tempSuccess...)
			validEnergies = validEnergies[tokenCount:]
			leftAmount -= tempAmount

			/*select {
			case <- endTimer.C:
				break loop
			default:*/
				
				
				/*if(tokenNum > len(validEnergies)){
					bidNum = len(validEnergies)
				}else {
					bidNum = tokenNum
				}
				fmt.Printf("max:%d\n", bidNum)*/

			// tempSuccess := bid(contract, validEnergies, tokenCount, input)

			

			// tokenNum -= len(tempSuccess)
			
		}
	
	data.BidAmount = 0
	data.BidSolar = 0
	data.BidWind = 0 
	data.BidThermal = 0
	// data.LatestAuctionStartTime = 0
	data.LastBidTime = 0
	data.FirstBidTime = time.Now().UnixNano()

	for i := 0; i < len(success); i++ {
		data.BidAmount += success[i].Amount
		switch success[i].SmallCategory {
		case "solar":
			data.BidSolar += success[i].Amount
		case "wind":
			data.BidWind += success[i].Amount
		case "thermal":
			data.BidThermal += success[i].Amount
		}
		/*if (data.LatestAuctionStartTime < success[i].AuctionStartTime) {
			data.LatestAuctionStartTime = success[i].AuctionStartTime
		}*/
		if (data.LastBidTime < success[i].BidTime) {
			data.LastBidTime = success[i].BidTime
		} else if (data.FirstBidTime > success[i].BidTime) {
			data.FirstBidTime = success[i].BidTime
		}
	}
	// fmt.Printf("%s bid return : %fWh\n", data.UserName, data.BidAmount)
	return success, data, nil
	

	// return successList, autcionStartMin, err
}

/*func bidold(contract *client.Contract, energies []Energy, tokenCount int, data Data) ([]Energy, float64) {
	successEnergy := []Energy{}
	//leftEnergy := energies
	
	//c := make(chan Energy)
	var wg sync.WaitGroup
	for i := 0; i < tokenCount; i++ {
		wg.Add(1)
		go func(i int){
			defer wg.Done()
			// fmt.Printf("id:%s, auctionStartTime:%v\n", energies[i].ID, time.Unix(energies[i].AuctionStartTime, 0))
			// id string, newOwner string, newBidPrice float64, priority float64, amount float64, timestamp int64, newID string) (*Output, error) {
			// var output Output

			message, timestamp, bidId, err := bidOnEnergy(contract, energies[i].ID, energies[i].BidPrice, data.UserName, data.BatteryLife, energies[i].Amount, energies[i].LargeCategory, energies[i].SmallCategory, energies[i].UnitPrice)
			if err != nil {
				log.Printf("%s function bid error:%v, timestamp:%v, now:%v\n", bidId, err, timestamp,  ((time.Now().Unix() - Diff - StartTime) * Speed + StartTime))
				energies[i].Error = "bidOnTokenError: " + err.Error()
				energies[i].Status = "F"
				return
				//c <- energies[i]
			}else if (message == "your bid is accepted") {
				// log.Printf("%s bid output: %s, timestamp:%v, now:%v\n", bidId, message, timestamp, ((time.Now().Unix() - Diff - StartTime) * Speed + StartTime))
				energies[i].EnergyID = energies[i].ID
				energies[i].ID = bidId
				energies[i].BidTime = timestamp
				energies[i].Status = "S"

			} else {
				// fmt.Printf("else: %s\n", output.Message)
				energies[i].Error = "OK"
				energies[i].Status = "F"
				// c <- energies[i]
			}
		}(i)
	}
	wg.Wait()

	var successAmount float64
	for i := 0; i < tokenCount; i++ {
		// energy := <-c
		if (energies[i].Status == "S") {
			successEnergy = append(successEnergy, energies[i])
			successAmount += energies[i].Amount
		}
	}

	// fmt.Println(successEnergy)

	return successEnergy, successAmount
}*/

func bid(contract *client.Contract, energies []Energy, tokenCount int, data Data) ([]Energy, float64) {
	successEnergy := []Energy{}
	var successAmount float64

	for i := 0; i < tokenCount; i++ {
		timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		energies[i].EnergyID = energies[i].ID
		energies[i].ID = fmt.Sprintf("%v%s%s-%d", timestamp, energies[i].EnergyID, data.UserName, r.Intn(10000))
		energies[i].Owner = data.UserName
		energies[i].BidAmount = energies[i].Amount
		energies[i].BidTime = timestamp
		energies[i].Priority = 1 - data.BatteryLife
	}
	input := energies[:tokenCount]
	idList, err := bidOnEnergy(contract, input)
	if err != nil {
		return successEnergy, 0
	}

	for i := 0; i < tokenCount; i++ {
		for _, id := range idList {
			if energies[i].ID == id {
				successEnergy = append(successEnergy, energies[i])
				successAmount += energies[i].Amount
				break
			} 
		}
	}

	return successEnergy, successAmount
}

func bidOnEnergy(contract *client.Contract, energies []Energy) ([]string, error) {
	var messageList []string
	energyJSON, err := json.Marshal(energies)
	if err != nil {
		panic(err)
	}
	//var out []BidReturn
	count := 0
	bidLoop:
	for {
		evaluateResult, err := contract.SubmitTransaction("BidOnEnergy", string(energyJSON))
		if len(evaluateResult) > 0 {
			message := string(evaluateResult)
			messageList = strings.Split(message, ",")
			messageList = messageList[:len(messageList) - 1]
			if err != nil {
				fmt.Println(err.Error())
			} 
			break bidLoop
		} else {
			count++
			if count > 3 {
				break bidLoop
				return messageList, fmt.Errorf("no bid")
			}
		}
	}
	
	
	return messageList, nil
}

// id string, newOwner string, newBidPrice float64, priority float64, amount float64, timestamp int64, newID string
//  energies[i].LargeCategory, energies[i].SmallCategory, energies[i].UnitPrice)
func bidOnEnergyold(contract *client.Contract, energyId string, bidPrice float64, username string, batteryLife float64, amount float64, lCat string, sCat string, unitPrice float64) (string, int64, string, error) {
	//fmt.Printf("Evaluate Transaction: BidOnToken, function returns asset attributes\n")
	// endTimer := time.NewTimer(time.Duration((EndTime - ((time.Now().Unix() - Diff - StartTime) * Speed + StartTime)) / Speed) * time.Second)

	var message string
	timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
	sTimestamp := fmt.Sprintf("%v", timestamp)
	sBidPrice := fmt.Sprintf("%v", bidPrice)
	sUnitPrice := fmt.Sprintf("%v", unitPrice)
	sPriority := fmt.Sprintf("%v", 1 - batteryLife)
	sAmount := fmt.Sprintf("%v", amount)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// create id
	bidId := fmt.Sprintf("%s%s%s-%d", sTimestamp, energyId, username, r.Intn(10000))

	// fmt.Printf("bid id:%s, timestamp:%s, price:%s\n", energyId, sTimestamp, sBidPrice)
	count := 0
	bidLoop:
	for {
		if ((time.Now().UnixNano() -Diff - StartTime) * Speed + StartTime > EndTime) {
			return "", timestamp, bidId, fmt.Errorf("time up")
		}
		/*select {
		case <- endTimer.C:
			return message, timestamp, bidId, fmt.Errorf("time up")
		default:*/
		isOk, err := bidOk(contract, energyId, bidPrice, (1 - batteryLife))
		if err != nil {
			// err
		}
		if isOk {
			submitResult, err := contract.SubmitTransaction("BidOnEnergy", bidId, energyId, username, sBidPrice, sPriority, sAmount, sTimestamp, lCat, sCat, sUnitPrice)
			if err != nil {
				//log.Printf("%s, bid error: %s, %v\n", username, energyId, err)
				panic(fmt.Errorf("failed to submit transaction: %s, bid error: %v, %v", username, energyId, err))
				// rand.Seed(time.Now().UnixNano())
				// timer := time.NewTimer(time.Duration(rand.Intn(1000000000)) * time.Nanosecond / time.Duration(Speed))
				count++
				if count > 3 {
					return "", timestamp, bidId, err
				}
				// <- timer.C
			} else {
				message = string(submitResult)
				// log.Printf("%s, bid on %s time: %s, now: %v, message: %s\n", username, energyId, sTimestamp, ((time.Now().Unix() - Diff - StartTime) * Speed + StartTime), message)
				break bidLoop
			}
		}
	}
	return message, timestamp, bidId, nil
}

//func (s *SmartContract) BidOk(ctx contractapi.TransactionContextInterface, energyId string, bidPrice float64, priority float64) (bool, error) {

	func bidOk(contract *client.Contract, energyId string, bidPrice float64, priority float64) (bool, error){
		sBidPrice := fmt.Sprintf("%v", bidPrice)
		sPriority := fmt.Sprintf("%v", priority)
		isOk := true

		for {
			if ((time.Now().UnixNano() -Diff - StartTime) * Speed + StartTime > EndTime) {
				return false, fmt.Errorf("time up")
			}
			evaluateResult, err := contract.EvaluateTransaction("BidOk", energyId, sBidPrice, sPriority)
			if err != nil {
				log.Printf("bid ok error:%v\n", err.Error())
				return true, nil
				//panic(fmt.Errorf("bid ok error failed to evaluate transaction: %v", err))
			} else {
				result := string(evaluateResult)
				isOk, err = strconv.ParseBool(result)
				if err != nil {
					log.Printf("parse string to bool error:%v\n", err.Error())
				}
				break
			}
		}

		return isOk, nil
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
	returnUpperLat := 2 * myLatitude - math.Abs(returnLowerLat) //緯度が0のとき、lowerLatがマイナスなため。日本は関係ないが。


	// fmt.Printf("lowerLat:%g\n", returnLowerLat)
	// fmt.Printf("upperLat:%g\n", returnUpperLat)
	// fmt.Printf("lowerLng:%g\n", returnLowerLng)
	// fmt.Printf("upperLng:%g\n", returnUpperLng)

	return returnLowerLat, returnUpperLat, returnLowerLng, returnUpperLng

}

func queryByLocationRange(contract *client.Contract, consumer string, lowerLat float64, upperLat float64, lowerLng float64, upperLng float64) ([]Energy, error) {
	strLowerLat := strconv.FormatFloat(lowerLat, 'f', -1, 64)
	strUpperLat := strconv.FormatFloat(upperLat, 'f', -1, 64)
	strLowerLng := strconv.FormatFloat(lowerLng, 'f', -1, 64)
	strUpperLng := strconv.FormatFloat(upperLng, 'f', -1, 64)

	// fmt.Printf("Async Submit Transaction: QueryByLocationRange'\n")

	result := []Energy{}
	for {
		if ((time.Now().UnixNano() -Diff - StartTime) * Speed + StartTime > EndTime) {
			return result, fmt.Errorf("time up")
		}
		evaluateResult, err := contract.EvaluateTransaction("QueryByLocationRange", "generated", consumer, strLowerLat, strUpperLat, strLowerLng, strUpperLng)
		if err != nil {
			log.Printf("queryByLocationRange error: %s, %v\n", consumer, err.Error())
			//panic(fmt.Errorf("queryByLocationRange error: %s, failed to evaluate transaction: %v", consumer, err))
		} else {
			// log.Printf("query success: %s\n", consumer)
			err = json.Unmarshal(evaluateResult, &result)
			if (err != nil && len(evaluateResult) > 0) {
				log.Printf("unmarshal error: %v\n", err.Error())
			} else {
				break
			}
		}
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