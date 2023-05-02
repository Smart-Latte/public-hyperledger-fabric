package producer

import (
	"fmt"
	"time"
	"math/rand"
	"math"
	"sync"
	
	"github.com/hyperledger/fabric-gateway/pkg/client"
)

func DummyWindProducer(contract *client.Contract, username string, lLat float64, uLat float64, lLon float64, uLon float64, category string, ratingOutput float64, ratingSpeed float64, 
	cutIn float64, outputList [dayNum][hourNum]float64, seed int64){

		r := rand.New(rand.NewSource(seed))
		lat := r.Float64() * (uLat - lLat) + lLat
		lon := r.Float64() * (uLon - lLon) + lLon
		fmt.Printf("%s, %g, %g\n", username, lat, lon)

		SeaWindProducer(contract, username, lat, lon, category, ratingOutput, ratingSpeed, cutIn, outputList, seed)

	}


func SeaWindProducer(contract *client.Contract, username string, lat float64, lon float64, category string, ratingOutput float64, ratingSpeed float64, cutIn float64, 
	speedList [dayNum][hourNum]float64, seed int64) {
	var outputList[dayNum][hourNum] float64

	for i := 0; i < dayNum; i++ {
		for j := 0; j < hourNum; j++ {
			if (speedList[i][j] >= cutIn) {
				outputList[i][j] = math.Pow((speedList[i][j] / ratingSpeed), 3) * ratingOutput
			}else {
				outputList[i][j] = 0
			}
		}
		fmt.Println(outputList[i])
	}
	Produce(contract, username, lat, lon, category, 1, outputList, seed)
	
}

func DummySolarProducer(contract *client.Contract, username string, lLat float64, uLat float64, lLon float64, uLon float64, category string, 
	output float64, outputList [dayNum][hourNum]float64, seed int64) {

		r := rand.New(rand.NewSource(seed))
		lat := r.Float64() * (uLat - lLat) + lLat
		lon := r.Float64() * (uLon - lLon) + lLon
		fmt.Printf("%s, %v, %g, %g\n", username, seed, lat, lon)

		Produce(contract, username, lat, lon, category, output, outputList, seed)
	
}

func Produce(contract *client.Contract, username string, lat float64, lon float64, category string, output float64, outputList [dayNum][hourNum]float64, seed int64) {

	end := time.Duration((EndTime - ((time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime)) / Speed)
	fmt.Printf("endTimer:%v\n", end)
	endTimer := time.NewTimer(end * time.Nanosecond)

	// output per min during un hour
	var myOutput[dayNum][hourNum] float64
	var timing[dayNum][hourNum] float64
	var wg sync.WaitGroup
	counter := 0

	var maxCreateInterval float64 = 2.5 // min
	var maxCreateAmount float64 = 2500 // Wh

	/*for i := 0; i <  dayNum; i++ {
		for j := 0; j < hourNum; j++ {
			fmt.Printf("%v ", output * outputList[i][j])
		}
		fmt.Println("")
	}*/

	for i := 0; i < dayNum; i++ {
		for j := 0; j < hourNum; j++ {
			outputPerHour := output * outputList[i][j]

			if (outputPerHour >= maxCreateAmount * 60 / maxCreateInterval) {
				myOutput[i][j] = maxCreateAmount
				timing[i][j] = 60 * 60 / (outputPerHour / maxCreateAmount)
			} else {
				myOutput[i][j] = outputPerHour / 60 * maxCreateInterval
				timing[i][j] = 60 * maxCreateInterval
			}
			fmt.Printf("%s, output:%v, timing:%v, ", username, myOutput[i][j], timing[i][j])
		}
	}
	
	r := rand.New(rand.NewSource(seed))
	wait := r.Intn(60) * 1000000000
	waitNano := r.Intn(1000000000)
	fmt.Printf("%s wait : %d, waitNano:%d\n", username, wait + 5, waitNano)
	timer := time.NewTimer((time.Duration(waitNano) * time.Nanosecond + time.Duration(5000000000 + int64(wait) + time.Now().UnixNano() - Diff - StartTime) * time.Nanosecond) / time.Duration(Speed))

	<- timer.C
	loop:
		for counter < 24 {
			thisTime := counter + StartHour
			var thisTiming float64
			var thisOut float64
			if thisTime < 24 {
				thisOut = myOutput[0][thisTime]
				thisTiming = timing[0][thisTime]
			} else if (thisTime < 48) {
				thisOut = myOutput[1][thisTime - 24]
				thisTiming = timing[1][thisTime - 24]
			} else {
				fmt.Println("simulation is too long")
			}
			ticker := time.NewTicker(time.Nanosecond * time.Duration(thisTiming * 1000000000) / time.Duration(Speed))
			// ticker := time.NewTicker(time.Second * time.Duration(thisTiming) / time.Duration(Speed))
			fmt.Printf("producer %v counter:%d, out:%v, timing:%v\n", username, counter, thisOut, thisTiming)
			thisTimeCounter := 0
			for {
				if (float64(thisTimeCounter) >= 60 * 60 / thisTiming) {
					ticker.Stop()
					fmt.Printf("producer %v counter:%d, this time counter:%d, %v\n", username, counter, thisTimeCounter, time.Unix(0, (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime))
					counter++
					break
				}
				// ログ
				select {
				case <-ticker.C:
					if (thisOut > 0) {
						timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
						var input Input = Input{User: username, Latitude: lat, Longitude: lon, Amount: thisOut, Category: category, Timestamp: timestamp}
						wg.Add(1)
						go func(i Input) {
							defer wg.Done()
							Create(contract, i)
						}(input)
					}
					// wg.Wait()
					thisTimeCounter++
				case <- endTimer.C:
					ticker.Stop()
					timestamp := (time.Now().UnixNano() - Diff - StartTime) * Speed + StartTime
					fmt.Printf("PRODUCER END TIMER: %v\n", time.Unix(0, timestamp))
					break loop
				}
				// Create
				// create counter + startHour
			// id string, latitude float64, longitude float64, producer string, amount float64, largeCategory string, smallCategory string, timestamp int64)
			}
		}
	fmt.Printf("%s finish\n", username)
	wg.Wait()
 }