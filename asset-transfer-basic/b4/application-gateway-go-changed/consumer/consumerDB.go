package consumer

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func DbResister(dataList [][][]Data) {
	db, err :=  sql.Open("sqlite3", "db/test1.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS "ConsumerData" ("ID" TEXT PRIMARY KEY, "UserName" TEXT, "Latitude" REAL, "Longitude" REAL, 
	"TotalAmountWanted" REAL, "HighPrice" INTEGER, "FirstBidTime" INTEGER, "LastBidTime" INTEGER, "BatteryLife" REAL, "Requested" REAL, 
	"BidAmount" REAL, "BidSolar" REAL, "BidWind" REAL, "BidThermal" REAL, 
	"GetAmount" REAL, "GetSolar" REAL, "GetWind" REAL, "GetThermal" REAL)`)
	if err != nil {
		panic(err)
	}

	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	stmt, err := tx.Prepare(`INSERT INTO ConsumerData (ID, UserName, Latitude, Longitude, TotalAmountWanted, 
		HighPrice, FirstBidTime, LastBidTime, BatteryLife, Requested, BidAmount, BidSolar, 
		BidWind, BidThermal, GetAmount, GetSolar, GetWind, GetThermal) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()


	for i := 0; i < len(dataList); i++ {
		for j := 0; j < len(dataList[i]); j++ {
			for k := 0; k < len(dataList[i][j]); k++ {
				id := fmt.Sprintf("%v%v%v-%v-%v", time.Now().Unix(), j, k, dataList[i][j][k].LastBidTime, dataList[i][j][k].UserName)
				_, err := stmt.Exec(id, dataList[i][j][k].UserName, dataList[i][j][k].Latitude, dataList[i][j][k].Longitude, 
					dataList[i][j][k].TotalAmountWanted, dataList[i][j][k].HighPrice, dataList[i][j][k].FirstBidTime, dataList[i][j][k].LastBidTime, 
					dataList[i][j][k].BatteryLife, dataList[i][j][k].Requested, dataList[i][j][k].BidAmount, dataList[i][j][k].BidSolar, 
					dataList[i][j][k].BidWind, dataList[i][j][k].BidThermal, dataList[i][j][k].GetAmount, dataList[i][j][k].GetSolar, 
					dataList[i][j][k].GetWind, dataList[i][j][k].GetThermal)
				if err != nil {
					fmt.Printf("i: %v, j: %v, k: %v, id: %v\n", i, j, k, id)
					panic(err)
				}
			}
			
		}

	}
	tx.Commit()
	//tx.Rollback()

	rows, err := db.Query(
		`SELECT * FROM ConsumerData`,
	)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var d Data
		err := rows.Scan(&d.ID, &d.UserName, &d.Latitude, &d.Longitude, &d.TotalAmountWanted, &d.HighPrice, &d.FirstBidTime, &d.LastBidTime, &d.BatteryLife, 
			&d.Requested, &d.BidAmount, &d.BidSolar, &d.BidWind, &d.BidThermal, &d.GetAmount, &d.GetSolar, &d.GetWind, &d.GetThermal)
		if err != nil {
			log.Println(err)
			return
		}
		//fmt.Println(d)
	}
}
