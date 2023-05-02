package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
	contractapi.Contract
}

// Asset describes basic details of what makes up a simple asset
// Insert struct field in alphabetic order => to achieve determinism across languages
// golang keeps the order when marshal to json but doesn't order automatically
type Energy struct {
	DocType          string    `json:"DocType`
	UnitPrice        float64   `json:"Unit Price"`
	BidPrice         float64   `json:"Bid Price"`
	GeneratedTime    time.Time `json:"Generated Time"`
	AuctionStartTime time.Time `json:"Auction Start Time"`
	BidTime          time.Time `json:"Bid Time"`
	ID               string    `json:"ID"`
	LargeCategory    string    `json:"LargeCategory"`
	Latitude         float64   `json:"Latitude"`
	Longitude        float64   `json:"Longitude"`
	Owner            string    `json:"Owner"`
	Producer         string    `json:"Producer"`
	SmallCategory    string    `json:"SmallCategory"`
	Status           string    `json:"Status"`
}

// InitLedger adds a base set of assets to the ledger
// Owner: Brad, Jin Soo, Max, Adriana, Michel
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	// var time1 = time.Date(2022, 11, 6, 16, 0, 0, 0, time.Local)
	/*timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil{
		return err
	}
	time := time.Unix(timestamp.Seconds, int64(timestamp.Nanos))*/

	energies := []Energy{
		{DocType: "cost", ID: "solar-power-cost", UnitPrice: 0.02,
			LargeCategory: "green", SmallCategory: "solar"},
		{DocType: "cost", ID: "wind-power-cost", UnitPrice: 0.02,
			LargeCategory: "green", SmallCategory: "wind"},
		{DocType: "cost", ID: "thermal-power-cost", UnitPrice: 0.03,
			LargeCategory: "depletable", SmallCategory: "thermal"},
		/*{DocType:"token", ID: "energy1", LargeCategory: "green", SmallCategory: "solor", Status: "generated", Producer: "User1", Owner: "User1",
		Latitude: 35.547766481196525, Longitude: 39.67124467488006, UnitPrice: 0.02, BidPrice: 0.02, GeneratedTime: time1, AuctionStartTime: time},
		{DocType:"token", ID: "energy2", LargeCategory: "green", SmallCategory: "solor", Status: "generated", Producer: "User2", Owner: "User2",
		Latitude: 35.54687478299901, Longitude: 139.67115184675248, UnitPrice: 0.02, BidPrice: 0.02, GeneratedTime: time1, AuctionStartTime: time1},
		{DocType:"token", ID: "energy3", LargeCategory: "green", SmallCategory: "solor", Status: "sold", Producer: "User3", Owner: "User1",
		Latitude: 35.54687478299901, Longitude: 139.67115184675248},
		{DocType:"token", ID: "energy4", LargeCategory: "green", SmallCategory: "solor", Status: "generated", Producer: "User4", Owner: "User4", 
		Latitude: 35.5552824466371, Longitude: 139.65527497388206, UnitPrice: 0.02, BidPrice: 0.02, GeneratedTime: time1, AuctionStartTime: time1},
		{DocType:"token", ID: "energy5", LargeCategory: "green", SmallCategory: "solor", Status: "generated", Producer: "User1", Owner: "User1",
		Latitude: 35.547766481196525, Longitude: 139.67124467488006, UnitPrice: 0.02, BidPrice: 0.02, GeneratedTime: time1, AuctionStartTime: time1},
		{DocType:"token", ID: "energy6", LargeCategory: "green", SmallCategory: "solor", Status: "generated", Producer: "User5", Owner: "User5", 
		Latitude: 35.64914672135123, Longitude: 139.7429409664394, UnitPrice: 0.02, BidPrice: 0.02, GeneratedTime: time1, AuctionStartTime: time1},*/
	}

	for _, energy := range energies {
		energyJSON, err := json.Marshal(energy)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(energy.ID, energyJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state. %v", err)
		}
	}

	return nil
}

func (s *SmartContract) UpdateUnitPrice(ctx contractapi.TransactionContextInterface, 
	smallCategory string, newUnitPrice float64, timestamp time.Time) error {
		var id = smallCategory + "-power-cost"
		cost, err := s.ReadToken(ctx, id)
		if err != nil {
			return err
		}
		cost.UnitPrice = newUnitPrice
		cost.GeneratedTime = timestamp

		costJSON, err := json.Marshal(cost)
			if err != nil {
				return err
			}

		err = ctx.GetStub().PutState(id, costJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state. %v", err)
		}
		return nil
}

func (s *SmartContract) DiscountUnitPrice(ctx contractapi.TransactionContextInterface, id string) (error) {
		energy, err := s.ReadToken(ctx, id)
		if err != nil {
			return err
		}
		energy.UnitPrice = energy.UnitPrice * 0.8

		energyJSON, err := json.Marshal(energy)
			if err != nil {
				return err
			}

		return ctx.GetStub().PutState(id, energyJSON)
}

// CreateAsset issues a new asset to the world state with given details.
// 新しいトークンの発行
// errorは返り値の型
// 引数は、ID、緯度、経度、エネルギーの種類、発電した時間、発電者、価格
// トークンには、オーナー、ステータスも含める
func (s *SmartContract) CreateToken(ctx contractapi.TransactionContextInterface,
	id string, latitude float64, longitude float64, producer string, largeCategory string, smallCategory string, timestamp time.Time) error {

	var costId = smallCategory + "-power-cost"

	cost, err := s.ReadToken(ctx, costId)
	if err != nil {
		return err
	}

	exists, err := s.EnergyExists(ctx, id)

	//get unit price

	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the energy %s already exists", id)
	}

	energy := Energy{
		DocType:          "token",
		ID:               id,
		Latitude:         latitude,
		Longitude:        longitude,
		Owner:            producer,
		Producer:         producer,
		LargeCategory:    largeCategory,
		SmallCategory:    smallCategory,
		Status:           "generated",
		GeneratedTime:    timestamp,
		AuctionStartTime: timestamp,
		UnitPrice:        cost.UnitPrice,
		BidPrice:         cost.UnitPrice,
	}
	energyJSON, err := json.Marshal(energy)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, energyJSON)
}

// TransferAsset updates the owner field of asset with given id in world state, and returns the old owner.
// 購入する
func (s *SmartContract) BidOnToken(ctx contractapi.TransactionContextInterface, id string, newOwner string, newBidPrice float64, timestamp time.Time) (string, error) {
	energy, err := s.ReadToken(ctx, id)
	if err != nil {
		return "", err
	}
	var returnMessage string
	//generatedTime := energy.GeneratedTime
	var generatedTimeCompare = timestamp.Add(time.Minute * -30)
	var auctionStartTimeCompare = timestamp.Add(time.Minute * -5)

	if generatedTimeCompare.After(energy.GeneratedTime) == true {
		returnMessage = "the energy " + id + " was generated more than 30min ago"
	}else if auctionStartTimeCompare.After(energy.AuctionStartTime) == true {
		returnMessage = "the auction of energy " + id + " was started more than 5min ago"
	} else {
		if energy.BidPrice >= newBidPrice {
			returnMessage = "your bid price is cheap"
		}else{
			// energy.Status = "sold"
			energy.BidTime = timestamp
			energy.Owner = newOwner
			energy.BidPrice = newBidPrice
			energyJSON, err := json.Marshal(energy)
			if err != nil {
				return "", err
			}

			err = ctx.GetStub().PutState(id, energyJSON)
			if err != nil {
				return "", err
			}
			returnMessage = "your bid was successful"
		}
	}

	/*if energy.Status == "generated" {
		energy.Status = "sold"
	} else {
		return "", fmt.Errorf("the energy %s is not for sale", id)
	}*/

	// var user []byte
	// user, err = (ctx.GetStub().GetCreator())
	// oldOwner = string(user)

	return returnMessage, nil
}

func (s *SmartContract) AuctionEnd(ctx contractapi.TransactionContextInterface, id string, producer string, timestamp time.Time) (string, error) {
	energy, err := s.ReadToken(ctx, id)

	var returnMessage string
	var generatedTimeCompare = timestamp.Add(time.Minute * -30)
	var auctionStartTimeCompare = timestamp.Add(time.Minute * -5)

	if err != nil {
		return "", err
	}

	if energy.GeneratedTime.After(generatedTimeCompare) == false {
		if energy.Owner == energy.Producer {
			energy.Status = "old"
			returnMessage = "the energy " + id + " was generated more than 30min ago. This was not sold."
		}else{
			energy.Status = "sold"
			returnMessage = "the energy " + id + " was sold. It was generetad more than 30min ago."
		}
	}else{
		if energy.AuctionStartTime.After(auctionStartTimeCompare) == false {
			if energy.Owner == energy.Producer {
				energy.AuctionStartTime = timestamp
				returnMessage = "the energy " + id + " was generated more than 5min ago. The Action Start Time was updated."
			}else{
				energy.Status = "sold"
				returnMessage = "the energy " + id + " was sold"
			}
		}else{
			returnMessage = ("Why did you call this function?")
		}
	}

	err = s.UpdateToken(ctx, energy)
	if err != nil {
		return "", err
	}
	return returnMessage, nil
}

// AssetExists returns true when asset with given ID exists in world state
// スタブの意味はよく分からない。台帳にアクセスするための関数らしい。一般的には「外部プログラムとの細かなインターフェース制御を引き受けるプログラム」を指すらしい
func (s *SmartContract) EnergyExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	energyJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return energyJSON != nil, nil
}

// ReadToken returns the asset stored in the world state with given id.
// トークンを返す
func (s *SmartContract) ReadToken(ctx contractapi.TransactionContextInterface, id string) (*Energy, error) {
	energyJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if energyJSON == nil {
		return nil, fmt.Errorf("the energy %s does not exist", id)
	}

	var energy Energy
	err = json.Unmarshal(energyJSON, &energy)
	if err != nil {
		return nil, err
	}

	return &energy, nil
}

func (s *SmartContract) UpdateToken(ctx contractapi.TransactionContextInterface, energy *Energy) error {
	energyJSON, err := json.Marshal(energy)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(energy.ID, energyJSON)

}

/*
// UpdateAsset updates an existing asset in the world state with provided parameters.
// 内容は読み込まない。存在することを確認し、上書きする
// ステータスを変更するため改造
func (s *SmartContract) UpdateAsset(ctx contractapi.TransactionContextInterface, id string) error {
	exists, err := s.EnergyExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the energy %s does not exist", id)
	}

	// overwriting original asset with new asset
	energy := Energy{
		ID:     id,
		Status: "old",
	}
	energyJSON, err := json.Marshal(energy)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, energyJSON)
}*/

func (s *SmartContract) QueryByStatus(ctx contractapi.TransactionContextInterface, status string) ([]*Energy, error) {
	queryString := fmt.Sprintf(`{"selector":{"DocType":"token","Status":"%s"},"use_index":["_design/indexStatusDoc","indexStatus"]}`, status)
	// queryString := fmt.Sprintf(`{"selector":{"docType":"asset","owner":"%s"}}`, owner)

	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var energies []*Energy
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var energy Energy
		err = json.Unmarshal(queryResponse.Value, &energy)
		if err != nil {
			return nil, err
		}
		energies = append(energies, &energy)
	}

	return energies, nil
}

func (s *SmartContract) QueryByLocationRange(ctx contractapi.TransactionContextInterface,
	status string, latitudeLowerLimit float64, latitudeUpperLimit float64,
	longitudeLowerLimit float64, longitudeUpperLimit float64) ([]*Energy, error) {

	queryString := fmt.Sprintf(`{"selector":{"DocType":"token","Status":"%s",
	"Latitude":{"$gte":%f,"$lte":%f},"Longitude":{"$gte":%f,"$lte":%f}},"use_index":["_design/indexLocationDoc","indexLocation"]}`,
		status, latitudeLowerLimit, latitudeUpperLimit, longitudeLowerLimit, longitudeUpperLimit)
		/*queryString := fmt.Sprintf(`{"selector":{"DocType":"token","Status":"%s",
		"Latitude":{"$gte":%f},"Longitude":{"$gte":%f}},"use_index":["_design/indexLocationDoc","indexLocation"]}`,
			status, latitudeLowerLimit, longitudeLowerLimit)*/
	// queryString := fmt.Sprintf(`{"selector":{"docType":"asset","owner":"%s"}}`, owner)
	//{ 'number' : {'$gte' :2, '$lte' : 3}

	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var energies []*Energy
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var energy Energy
		err = json.Unmarshal(queryResponse.Value, &energy)
		if err != nil {
			return nil, err
		}
		energies = append(energies, &energy)
	}

	return energies, nil
}

// GetAllAssets returns all assets found in world state
func (s *SmartContract) GetAllTokens(ctx contractapi.TransactionContextInterface) ([]*Energy, error) {
	// range query with empty string for startKey and endKey does an
	// open-ended query of all assets in the chaincode namespace.
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var energies []*Energy
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var energy Energy
		err = json.Unmarshal(queryResponse.Value, &energy)
		if err != nil {
			return nil, err
		}
		energies = append(energies, &energy)
	}

	return energies, nil
}

/*
// DeleteAsset deletes an given asset from the world state.
// 後回し？そもそもいらない？30分経ったものを消去するかどうか。ステータスを変更するにとどめるか
// 現状はUpdateAssetでステータスを変更
func (s *SmartContract) DeleteAsset(ctx contractapi.TransactionContextInterface, id string) error {
	exists, err := s.EnergyExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the energy %s does not exist", id)
	}

	return ctx.GetStub().DelState(id)
}*/
