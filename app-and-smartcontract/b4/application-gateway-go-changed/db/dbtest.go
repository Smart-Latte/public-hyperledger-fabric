package main
import (
	"database/sql"
	"fmt"
	"log"
	"time"
	//"os"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	Name string
	Age int
}


type Energy struct {
	DocType          string    `json:"DocType"`
	Amount float64 `json:"Amount"`
	BidAmount float64 `json:"BidAmount"`
	SoldAmount float64 `json:"SoldAmount"`
	UnitPrice        float64   `json:"Unit Price"`
	BidPrice         float64   `json:"Bid Price"`
	GeneratedTime    int64 `json:"Generated Time"`
	BidTime          int64 `json:"Bid Time"`
	ID               string    `json:"ID"`
	EnergyID string `json:"EnergyID"`
	LargeCategory    string    `json:"LargeCategory"`
	Latitude         float64   `json:"Latitude"`
	Longitude        float64   `json:"Longitude"`
	Owner            string    `json:"Owner"`
	Producer         string    `json:"Producer"`
	Priority float64 `json:"Priority"`
	SmallCategory    string    `json:"SmallCategory"`
	Status           string    `json:"Status"`
	Error string `json:"Error"`
}

type Data struct {
	ID int
	UserName string
	Latitude float64
	Longitude float64
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


func main() {
	/*_, err := os.Stat("db")
	if err != nil {
		panic(err)
	}
	fmt.Println("../db")
	_, err = os.Stat("../db/test.db")
	if err != nil {
		panic(err)
	}
	fmt.Println("../db/test.db")*/


	db, err :=  sql.Open("sqlite3", "db/test0.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS "Data" ("ID" INTEGER PRIMARY KEY, "UserName" TEXT, "Latitude" REAL, "Longitude" REAL, "TotalAmountWanted" REAL, 
	"FirstBidTime" INTEGER, "LastBidTime" INTEGER, "BatteryLife" REAL, "Requested" REAL, "BidAmount" REAL, "BidSolar" REAL, "BidWind" REAL, "BidThermal" REAL, 
	"GetAmount" REAL, "GetSolar" REAL, "GetWind" REAL, "GetThermal" REAL)`)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS "User" ("Name" STRING PRIMARY KEY, "Age" INTEGER)`)
	if err != nil {
		panic(err)
	}

	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	stmt, err := tx.Prepare(`INSERT INTO Data (ID, UserName, Latitude, Longitude, TotalAmountWanted, FirstBidTime, LastBidTime, BatteryLife, Requested, BidAmount, BidSolar, 
		BidWind, BidThermal, GetAmount, GetSolar, GetWind, GetThermal) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("%d%d", time.Now().Unix(), i)
		username := fmt.Sprintf("user%d", i)
		lat := float64(i) * 1.1
		lon := float64(i) * 1.1
		wanted := float64(i) * 10000.111
		first := i * 20000
		last := i * 20000
		life := float64(i) * 0.9
		req := float64(i) * 1000.1
		amount := float64(i) * 500.8
		solar := float64(i) * 200.8
		wind := float64(i) * 300

		_, err := stmt.Exec(id, username, lat, lon, wanted, first, last, life, req, amount, solar, wind, 0, amount, solar, wind, 0)
		if err != nil {
			panic(err)
		}
	}
	isOk := true

	stmt, err = tx.Prepare(`INSERT INTO User (Name, Age) VALUES (?, ?)`)
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("user%d", i)
		age := i
		_, err := stmt.Exec(name, age)
		if err != nil {
			panic(err)
		}
	}



	if isOk {
		tx.Commit()
	} else {
		tx.Rollback()
	}


	rows, err := db.Query(
		`SELECT UserName, GetAmount FROM Data`,
	)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var d Data
		err := rows.Scan(&d.UserName, &d.GetAmount)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Printf("username: %v, getAmount: %v\n", d.UserName, d.GetAmount)
	}

	rows, err = db.Query(
		`SELECT * FROM User`,
	)
	for rows.Next() {
		var user User
		err := rows.Scan(&user.Name, &user.Age)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Printf("name: %v, age: %v\n", user.Name, user.Age)
	}
}
