package main

import (
	// "fmt"
	"time"
	"sync"
	"math"
	oprt "github.com/Smart-Latte/fabric-samples/blockchain-application/b4/proposal/operator"
	prdc "github.com/Smart-Latte/fabric-samples/blockchain-application/b4/proposal/producer"
	cnsm "github.com/Smart-Latte/fabric-samples/blockchain-application/b4/proposal/consumer"
)

const dayNum = 2
const hourNum = 24

var tempData[dayNum][hourNum] float64 = [dayNum][hourNum]float64 {
	{87, 84, 85, 84, 83, 84, 90, 97, 109, 117, 121, 126, 130, 127, 125, 118, 116, 106, 100, 94, 92, 89, 88, 87}, 
	{85, 82, 82, 80, 76, 69, 81, 101, 112, 121, 130, 145, 140, 141, 143, 140, 128, 114, 106, 99, 94, 90, 88, 86}}
var insolation[dayNum][hourNum]  float64 =[dayNum][hourNum]float64 {
	{0, 0, 0, 0, 0, 3, 15, 52, 221, 293, 343, 366, 360, 320, 250, 114, 75, 9, 0, 0, 0, 0, 0, 0}, 
	{0, 0, 0, 0, 0, 3, 23, 99, 130, 214, 193, 319, 343, 309, 260, 156, 48, 8, 0, 0, 0, 0, 0, 0}}
var windSpeed[dayNum][hourNum]  float64 =[dayNum][hourNum]float64 {
	{6.6, 5.2, 6.2, 5.6, 5.7, 5.3, 5.6, 7.2, 7.4, 7.8, 8.6, 9.4, 8.4, 9.0, 6.9, 6.7, 5.7, 3.9, 3.4, 4.5, 4.4, 3.4, 4.1, 3.8}, 
	{3.8, 3.0, 4.4, 3.5, 2.0, 1.5, 2.1, 4.2, 3.9, 4.4, 4.5, 5.2, 5.4, 5.1, 6.1, 3.9, 3.8, 3.6, 2.9, 2.6, 2.4, 2.2, 1.8, 2.2}}
var bigSolarOutput[dayNum][hourNum] float64
var houseSolarOutput[dayNum][hourNum] float64
var landWindSpeed[dayNum][hourNum] float64
var seaWindSpeed[dayNum][hourNum] float64

func main() {
	for i := 0; i < dayNum; i++ {
		for j := 0; j < hourNum; j++ {
			bigSolarOutput[i][j] = 0.97 * 0.95 * 0.94 * 0.97 * 0.9 * (1 - 0.45 * (tempData[i][j] * 0.1 + 18.4 - 25) / 100) * (insolation[i][j] * 10 / 3.6 / 1000)
			houseSolarOutput[i][j] = 0.97 * 0.95 * 0.94 * 0.97 * 0.9 * (1 - 0.45 * (tempData[i][j] * 0.1 + 21.5 - 25) / 100) * (insolation[i][j] * 10 / 3.6 / 1000)
			seaWindSpeed[i][j] = windSpeed[i][j] * math.Pow((82 / 19), 0.1)
		}
	}
	startHour := 12
	startTime := time.Date(2015, time.March, 27, startHour, 0, 0, 0, time.Local).UnixNano()
	nowTime := time.Now().UnixNano()
	diff := nowTime - startTime
	endTime := time.Date(2015, time.March, 27, startHour + 24, 0, 0, 0, time.Local).UnixNano()
	var interval int64 = 1
	var tokenLife int64 = 30
	var speed int64 = 3
	var wg sync.WaitGroup

	oprt.InitOperator()
	wg.Add(3)
	go func() {
		defer wg.Done()
		oprt.Operator(startTime, endTime, diff, speed, bigSolarOutput, windSpeed, startHour)
	}()
	go func() {
		defer wg.Done()
		prdc.AllProducers(startTime, endTime, diff, speed, interval, tokenLife, bigSolarOutput, houseSolarOutput, windSpeed, seaWindSpeed, startHour)
	}()
	go func() {
		defer wg.Done()
		// start int64, end int64, diff int64, auctionSpeed int64, auctionInterval int64, life int64, startHour int
		cnsm.AllConsumers(startTime, endTime, diff, speed, interval, tokenLife, startHour)
	}()
	go func() {
		http.HandleFunc("/purchase", cnsm.ConsumerHandler)
		http.HandleFunc("/generate", prdc.ProducerHandler)
		http.ListenAndServe(":9080", nil)
	}
	wg.Wait()
}