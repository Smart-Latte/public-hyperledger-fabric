package consumer

import (
	"fmt"
	//"io/ioutil"
	//"log"
	//"path"
	"time"
	"math/rand"
	"github.com/hyperledger/fabric-gateway/pkg/client"
)

/* 
複数の需要家
type1: 普通充電 昼に充電
type2: 普通充電 夜に充電
type3: 急速充電 昼に充電
*/

// ゴールーチンで各ユーザ起動
// input: シミュレーション開始時間

type Data struct {
	ID string
	UserName string
	Latitude float64
	Longitude float64
	HighPrice int
	TotalAmountWanted float64
	FirstBidTime int64
	LastBidTime int64
	BatteryLife float64
	Requested float64
	BidAmount float64
	BidSolar float64
	BidWind float64
	BidThermal float64
	GetAmount float64
	GetSolar float64
	GetWind float64
	GetThermal float64
}

/*type Input struct {
	UserName string
	Latitude float64
	Longitude float64
	Amount float64
	BatteryLife float64
}

type ResultInput struct {
	UserName string
	StartTime int64
	EndTime int64
}*/

func Morning(contract *client.Contract, peer string, lLat float64, uLat float64, lLon float64, uLon float64, battery float64, chargeTime float64, seed int64) []Data {
	username := fmt.Sprintf("%vmorningUser%v", peer, seed)
	r := rand.New(rand.NewSource(seed))
	add := r.Intn(4)
	DataList := Consume(contract, username, lLat, uLat, lLon, uLon, time.Duration(add), battery, chargeTime, 1, seed)
	return DataList
}

func Night(contract *client.Contract, peer string, lLat float64, uLat float64, lLon float64, uLon float64, battery float64, chargeTime float64, seed int64) []Data {
	username := fmt.Sprintf("%vnightUser%v", peer, seed)
	r := rand.New(rand.NewSource(seed))
	add := 12 + r.Intn(4)
	DataList := Consume(contract, username, lLat, uLat, lLon, uLon, time.Duration(add), battery, chargeTime, 1, seed)
	return DataList
}

func Fast(contract *client.Contract, peer string, lLat float64, uLat float64, lLon float64, uLon float64, battery float64, chargeTime float64, seed int64) []Data {
	username := fmt.Sprintf("%vfastUser%v" ,peer, seed)
	r := rand.New(rand.NewSource(seed))
	add := r.Intn(11)
	DataList := Consume(contract, username, lLat, uLat, lLon, uLon, time.Duration(add), battery, chargeTime, 0.8, seed)
	return DataList
}

func General(contract *client.Contract, peer string, lLat float64, uLat float64, lLon float64, uLon float64, battery float64, chargeTime float64, seed int64) []Data {
	username := fmt.Sprintf("generalUser%v", seed)
	r := rand.New(rand.NewSource(seed))
	add := r.Intn(24)
	DataList := Consume(contract, username, lLat, uLat, lLon, uLon, time.Duration(add), battery, chargeTime, 1, seed)

	return DataList
}


// 充電開始時間(差分)、バッテリー容量(Wh)、チャージ済み(Wh)、充電時間(hour)、最終的なバッテリー残量(0から1)
func Consume(contract *client.Contract, username string, lLat float64, uLat float64, lLon float64, uLon float64, add time.Duration, battery float64, chargeTime float64, finalLife float64, seed int64) []Data {

	endTimer := time.NewTimer(time.Duration((EndTime - ((time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime)) / Speed) * time.Nanosecond)

	r := rand.New(rand.NewSource(seed))
	wait := time.Hour * add + time.Minute * time.Duration(1 + r.Intn(60)) + time.Second * time.Duration(r.Intn(60))
	waitNano := time.Nanosecond * time.Duration(r.Intn(1000000000))
	
	//fmt.Printf("%s wait : %v, waitNano:%d\n", username, wait, waitNano)

	timer := time.NewTimer((waitNano + wait) / time.Duration(Speed))
	lat := r.Float64() * (uLat - lLat) + lLat
	lon := r.Float64() * (uLon - lLon) + lLon
	//fmt.Printf("lat: %g, lon: %g\n", lat, lon)

	var charged float64
	charged = float64(r.Intn(int(battery * finalLife)))

	consumeData := []Data{}
	//input := Input{UserName: username, Latitude: lat, Longitude: lon}
	chargedEnergy := charged
	finalBattery := battery * finalLife
	require := finalBattery - charged
	chargePerSec := battery / chargeTime / 60 / 60
	requestMax := battery * finalLife / chargeTime / (60 / float64(TokenLife))
	//fmt.Println(requestAmount)
	var beforeUse float64 = 0
	var err error
	highLow := 0

	select {
	case <- endTimer.C:
		//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
		//fmt.Printf("CONSUMER END TIMER new: %v\n", time.Unix(0, timestamp))
	case <- timer.C:
	}

	// ticker := time.NewTicker(time.Duration(Interval) * time.Minute / time.Duration(Speed))
	zeroCount := 0
	// 1回目: amountPerMin * 2入札
	// var getEnergy float64 = 0
	//prevDone := true
	//consumerStart := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
	loop:
		for i := 0; ; i++ {
			// bid
			// 
			// fmt.Printf("%s: cosumerAppCount: %d\n", username, i)
			var requestAmount float64
			if (chargedEnergy >= finalBattery) {
				// fmt.Println("charged")
				endTimer.Stop()
				break loop
			}
			if (requestMax < finalBattery - chargedEnergy) {
				requestAmount = requestMax - beforeUse
			} else {
				requestAmount = finalBattery - chargedEnergy
			}

			batteryLife := chargedEnergy / battery
			consumeData = append(consumeData, Data{UserName: username, BatteryLife: batteryLife, HighPrice: highLow, Requested: requestAmount, Latitude: lat, Longitude: lon, TotalAmountWanted: require})
			var bidEnergies []Energy

			select {
			case <- endTimer.C:
				//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
				//fmt.Printf("CONSUMER END TIMER new: %v\n", time.Unix(0, timestamp))
				break loop
			default:
				bidEnergies, consumeData[i], err = Bid(contract, consumeData[i])
				if err != nil {
					//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
					//fmt.Printf("CONSUMER END TIMER new: %v\n", time.Unix(0, timestamp))
					break loop
				}
				// fmt.Printf("%s bid energy: %vWh\n", username, consumeData[i].BidAmount)
			}

		// 	fmt.Println("next is result")
			if (consumeData[i].BidAmount == 0) {
				zeroCount++
				highLow = 1
				
				if (zeroCount > 10) {
					//fmt.Println("%s long NO BID", username)
					longZeroTimer := time.NewTimer(30 * 60 * 1000000000 * time.Nanosecond / time.Duration(Speed))
					select {
						case <- endTimer.C:
							//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
							//fmt.Printf("CONSUMER END TIMER: %v\n", time.Unix(0, timestamp))
							break loop
						case <- longZeroTimer.C:
							zeroCount = 0
							continue loop
					}
				} else {
					//fmt.Printf("%s, nothing, zeroCount:%v\n", username, zeroCount)
					nothingTimer := time.NewTimer(60 * 1000000000 * time.Nanosecond / time.Duration(Speed))
					select {
					case <- endTimer.C:
						//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
						//fmt.Printf("CONSUMER END TIMER: %v\n", time.Unix(0, timestamp))
						break loop
					case <- nothingTimer.C:
						continue loop
					}
				}
			} else {
				highLow = 0
				zeroCount = 0
			}

			checkTime := (consumeData[i].LastBidTime + (Interval * 60 + 30) * 1000000000)
			wait := (checkTime - ((time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime)) / Speed

			//fmt.Printf("%s check wait:%v sec\n", username, float64(wait) / 1000000000)

			if (wait > 0) {
				resultTimer := time.NewTimer(time.Duration(wait) * time.Nanosecond)
				select {
				case <- resultTimer.C:
					// fmt.Println("result Timer")
				case <- endTimer.C:
					//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
					//fmt.Printf("CONSUMER END TIMER: %v\n", time.Unix(0, timestamp))
					break loop
				}
			}
			// bidResult
			select{
			case <-endTimer.C:
				//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
				//fmt.Printf("CONSUMER END TIMER: %v\n", time.Unix(0, timestamp))
				break loop
			default:
				consumeData[i], err = BidResult(contract, bidEnergies, consumeData[i])
				if err != nil {
					//fmt.Printf("time up: %v\n", err)
					break loop
				} else {
					//log.Printf("%s: count: %d, bidEnergy:%g, getEnergy:%gWh\n", username, i, consumeData[i].BidAmount, consumeData[i].GetAmount)
				}
			}

			chargedEnergy += consumeData[i].GetAmount
			var nextBidWait float64
			if (consumeData[i].GetAmount >= chargePerSec * 60 * 5) {
				nextBidWait = ((consumeData[i].GetAmount + beforeUse) / chargePerSec) - 5 * 60
				beforeUse = chargePerSec * 60 * 5
			} else {
				nextBidWait = 0
			}
			nextBidTimer := time.NewTimer(time.Duration(nextBidWait * 1000000000) * time.Nanosecond / time.Duration(Speed))
			//fmt.Printf("%s next bid: after %v min\n", username, nextBidWait / 60 / float64(Speed))

			if consumeData[i].GetAmount == 0 {
				highLow = 1
			} else {
				highLow = 0
			}

			select {
			case <- nextBidTimer.C:
				break
			case <- endTimer.C:
				nextBidTimer.Stop()
				//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
				//fmt.Printf("CONSUMER END TIMER: %v\n", time.Unix(0, timestamp))
				break loop
			}
		}
	//timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
	//fmt.Printf("consumer %s finish start:%v, end:%v, finalBattery:%v, chargedEnergy:%v\n", username, time.Unix(0, consumerStart), time.Unix(0, timestamp), finalBattery, chargedEnergy)
	return consumeData

	
/*
	for i := 0; ; i++ {
		// <- ticker.C
		if (i > 1) {
			if (consumeData[i - 2].Done == false) {
				i--
				prevDone = false
				continue
			} else {
				chargedEnergy += consumeData[i - 2]
			}
		}

		batteryLife := chargedEnergy / battery
		fmt.Println(batteryLife)

		if chargedEnergy >= finalBattery || zeroCount == 3{
			ticker.Stop()
			fmt.Println("break")
			break
		}

		timestamp := ((time.Now().Unix() - Diff - StartTime) * Speed + StartTime)
		var requestAmount float64
		if (i > 0 && chargedEnergy + consumeData[i - 1].Reqeustd >= finalBattery) {
			requestAmount = 0
		} else if (i > 1) {
			if (prevDone == false) {
				requestAmount = amountPerMin * 6 - consumeData[i - 2].GetAmount * 2
				prevDone = true
			} else {
				requestAmount = amountPerMin * 3 - consumeData[i - 2].GetAmount
			}
		} else {
			requestAmount = amountPerMin * 3
		}
		
		consumeData.appned(consumeData, Data{UserName: username, Battery: batteryLife, Done: false})
		input.Timestamp = timestamp
		input.Amount = requestAmount
		input.BatteryLife = batteryLife

		// bid
		// append data. requestTime, battery, Requested

		type Data struct {
			UserName string
			FirstAuctionStartTime int64
			LatestAuctionStartTime int64
			Battery float64
			Requested float64
			BidAmount float64
			BidSolar float64
			BidWind float64
			BidThermal float64
			GetAmount float64
			GetSolar float64
			GetWind float64
			GetThermal float64
			Done bool
		}
		consumeData[i].FirstAuctionStartTime, consumeData[i].LatestAuctionStartTime, consumeData[i].Requested, consumeData[i].BidAmount, 
		consumeData[i].BidSolar, consumeData[i].BidWind, consumeData[i].BidThermal = Bid(contract, input)


		wg.Add(1)
			go func(input Input, int i) {
				defer wg.Done()
				timer := newTimer(time.Duration(Interval) * time.Minute / time.Duration(Speed))
				resultinput = ResultInput{UserName: username, StartTime: Data[i].FirstAuctionStartTime + Interval, EndTime: Data[i].LatestAuctionStartTime + Interval}
				<- timer.C
				consumeData[i].GetAmount, consumeData[i].GetSolar, consumeData[i].GetWind, 
				consumeData[i].GetThermal = auctionresult(resultInput)
				consumeData[i].Done = true
			}(input,  i)
		
	}
	wg.Wait()



	// getEnergy := bid(math.Ceil(amountPerMin * 2), lat, lon, username, batteryLife)
	if getEnergy == 0 {
		zeroCount++
		fmt.Printf("zeroCount: %d\n", zeroCount)
	} else {
		chargedEnergy += getEnergy
		zeroCount = 0
	}

	for {
		if chargedEnergy >= battery || zeroCount == 3 {
			ticker.Stop()
			fmt.Printf("break\n")
			break
		}
		// tickerではなく、getEnergy後に再計算
		// getEnergyが返ってくるまでにかかる時間は1分以上
		// 返ってくる前にtickerでやってもいい？前々回までのデータを使って次の入札をすることになる
		// [100, 200, 10, ]みたいに得られた電力量保存？
		// getEnergy := bid(math.Ceil(amountPerMin * 2), lat, lon, username, batteryLife)
		// ログ
		<-ticker.C
		getEnergy = 0
		if getEnergy == 0 {
			zeroCount++
		} else {
			zeroCount = 0
			chargedEnergy += getEnergy
		}
		fmt.Printf("zeroCount: %d\n", zeroCount)
	}*/

}
