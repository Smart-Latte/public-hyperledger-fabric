package chaincode

import (
	"encoding/json"
	"fmt"
	// "sort"
	// "errors"
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
	Distance float64 `json:"Distance"`
}

/*type Output struct {
	Message string `json:"Message"`
	Amount float64 `json:"Amount"`
}*/
type Input struct {
	ID string 	`json:"ID"`
	Amount float64 	`json:"Amount"`
	Time int64 `sjon:"Time"`
}
type BidReturn struct {
	ID string `json:"ID"`
	Message string `json:"Message"`
	Error error `json:"Error"`
}

const (
	tokenLife = 30 // minute
	auctionInterval = 1 // minute
)

// InitLedger adds a base set of assets to the ledger
// Owner: Brad, Jin Soo, Max, Adriana, Michel
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {

	energies := []Energy{
		{DocType: "cost", ID: "solar-power-cost", UnitPrice: 0.02,
			LargeCategory: "green", SmallCategory: "solar"},
		{DocType: "cost", ID: "wind-power-cost", UnitPrice: 0.02,
			LargeCategory: "green", SmallCategory: "wind"},
		{DocType: "cost", ID: "thermal-power-cost", UnitPrice: 0.02,//0.1,
			LargeCategory: "depletable", SmallCategory: "thermal"},
		
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
	smallCategory string, newUnitPrice float64, timestamp int64) error {
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

// CreateAsset issues a new asset to the world state with given details.
// 新しいトークンの発行
// errorは返り値の型
// 引数は、ID、緯度、経度、エネルギーの種類、発電した時間、発電者、価格
// トークンには、オーナー、ステータスも含める
func (s *SmartContract) CreateEnergyToken(ctx contractapi.TransactionContextInterface,
	id string, latitude float64, longitude float64, producer string, amount float64, largeCategory string, smallCategory string, timestamp int64) (*Energy, error) {
	var energy Energy
	var costId = smallCategory + "-power-cost"

	cost, err := s.ReadToken(ctx, costId)
	if err != nil {
		return &energy, err
	}

	exists, err := s.EnergyExists(ctx, id)

	//get unit price

	if err != nil {
		return &energy, err
	}
	if exists {
		return &energy, fmt.Errorf("the energy %s already exists", id)
	}
	
	energy = Energy{
		DocType:          "token",
		ID:               id,
		Latitude:         latitude,
		Longitude:        longitude,
		Producer:         producer,
		LargeCategory:    largeCategory,
		SmallCategory:    smallCategory,
		Amount: amount, 
		Status:           "generated",
		GeneratedTime:    timestamp,
		UnitPrice:        cost.UnitPrice,
	}
	energyJSON, err := json.Marshal(energy)
	if err != nil {
		return &energy, err
	}

	return &energy, ctx.GetStub().PutState(id, energyJSON)
}

// TransferAsset updates the owner field of asset with given id in world state, and returns the old owner.
// 購入する
/*func (s *SmartContract) BidOnEnergy(ctx contractapi.TransactionContextInterface, 
	bidId string, energyId string, bidder string, bidPrice float64, priority float64, amount float64, timestamp int64, lCat string, sCat string, unitPrice float64) (string, error) {
	
	exists, err := s.EnergyExists(ctx, bidId)

	//get unit price

	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("the energy %s already exists", energyId)
	}
	
	bid := Energy{
		DocType:          "bid",
		ID:               bidId,
		EnergyID: energyId,
		Owner:         	  bidder,
		LargeCategory:    lCat,
		SmallCategory:    sCat,
		BidAmount: amount,
		Status:           "bid",
		UnitPrice:        unitPrice,
		BidPrice:         bidPrice,
		Priority: 		  priority, 
		BidTime: timestamp,
	}

	bidJSON, err := json.Marshal(bid)
	if err != nil {
		return "", err
	}

	err = ctx.GetStub().PutState(bidId, bidJSON)
	if err != nil {
		return "", err
	}

	return "your bid is accepted", nil
}*/

func (s *SmartContract) BidOnEnergy(ctx contractapi.TransactionContextInterface, bidList []*Energy) (string, error) {
	// bidId string, energyId string, bidder string, bidPrice float64, priority float64, amount float64, timestamp int64, lCat string, sCat string, unitPrice float64
	// var bidReturn []*BidReturn
	var rerr error
	var message string
	var eMessage string

	for i := 0; i < len(bidList); i++ {
		energy, err := s.ReadToken(ctx, bidList[i].EnergyID)
		if err != nil {
			eMessage += fmt.Sprintf("%v,", err.Error())
			continue
		}
		if energy.LargeCategory != bidList[i].LargeCategory && energy.SmallCategory != bidList[i].SmallCategory {
			eMessage += fmt.Sprintf("%v,", err.Error())
			continue
		}
		exists, err := s.EnergyExists(ctx, bidList[i].ID)
		if err != nil {
			eMessage += fmt.Sprintf("%v,", err.Error())
			continue
		}
		if exists {
			eMessage += fmt.Sprintf("%v,", err.Error())
			continue
		}
		bidList[i].Amount = 0
		bidList[i].DocType = "bid"
		bidList[i].Status = "bid"

		bidJSON, err := json.Marshal(bidList[i])
		if err != nil {
			eMessage += fmt.Sprintf("%v,", err.Error())
			continue
		}

		err = ctx.GetStub().PutState(bidList[i].ID, bidJSON)
		if err != nil {
			eMessage += fmt.Sprintf("%v,", err.Error())
			continue
		}
		message += fmt.Sprintf("%v,", bidList[i].ID)

	}
	/*if len(bidReturn) != len(bidList) {
		//return bidReturn, fmt.Errorf("incorrect length")
		rerr = fmt.Errorf("incorrect length")
		
	}*/
	if len(eMessage) != 0 {
		rerr = fmt.Errorf(eMessage)
	}

	return message, rerr
}

func (s *SmartContract) ChangeToken(ctx contractapi.TransactionContextInterface, energy *Energy) (error) {
	energyJSON, err := json.Marshal(energy)
	if err != nil {
		return err
	}
	

	err = ctx.GetStub().PutState(energy.ID, energyJSON)
	if err != nil {
		return err
	}
	return nil
}

func (s *SmartContract) AuctionEnd(ctx contractapi.TransactionContextInterface, energyInput *Input, bidInput []*Input) (string, error) {
	var message string
	var bidList []*Energy
	energy, err := s.ReadToken(ctx, energyInput.ID)
	if err != nil {
		return "", err
	}
	generatedTimeCompare := energyInput.Time - 60 * tokenLife * 1000000000
	if (generatedTimeCompare > energy.GeneratedTime) {
		energy.Status = "old"
		message = "the energy was generated more than 30min ago. This was not sold."
	} else {
		if (energy.Amount < energy.SoldAmount + energyInput.Amount) {
			return "energy amount is wrong", nil
		}
		energy.SoldAmount += energyInput.Amount
		if (energy.Amount == energy.SoldAmount) {
			energy.Status = "sold"
			message = "auction end"
		} else {
			message = "auction continue"
		}
		
		for i := 0; i < len(bidInput); i++ {
			if (bidInput[i].ID == "old") {
				return "the energy is alive", nil
			}
			bid, err := s.ReadToken(ctx, bidInput[i].ID)
			if (err != nil) {
				return "", err
			}
			if (bid.EnergyID != energy.ID) {
				return "energy ID is wrong", nil
			}
			if (bidInput[i].Amount > bid.BidAmount) {
				return "bid amount is wrong", nil
			}
			bid.Amount = bidInput[i].Amount
			bid.Producer = energy.Producer
			bid.Status = "success"
			bidList = append(bidList, bid)
		}
	}
	err = s.UpdateToken(ctx, energy)
		if err != nil {
			return "", err
		}

	for _, bid := range bidList {
		err := s.UpdateToken(ctx, bid)
		if err != nil {
			return "", err
		}
	}

	return message , nil 
	
	


	/*newEnergy := energy

	startTime := timestamp - auctionInterval * 75
	endTime := timestamp - 1

	bidding, err := s.QueryAuctionEnd(ctx, energyId, startTime, endTime)
	if err != nil {
		return "", err
	}

	sort.SliceStable(bidding, func(i, j int) bool {
		return bidding[i].BidTime < bidding[j].BidTime
	})

	sort.SliceStable(bidding, func(i, j int) bool {
		return bidding[i].Priority > bidding[j].Priority
	})

	sort.SliceStable(bidding, func(i, j int) bool {
		return bidding[i].BidPrice > bidding[j].BidPrice
	})

	generatedTimeCompare := timestamp - 60 * tokenLife

	totalAmount := newEnergy.Amount - newEnergy.SoldAmount
	successList := []*Energy{}
	var returnMessage string
	test := ""

	if generatedTimeCompare  >= newEnergy.GeneratedTime && len(bidding) == 0 {
		newEnergy.Status = "old"
		returnMessage = "the energy was generated more than 30min ago. This was not sold."
	} else {
		for _, b := range bidding {
			if (totalAmount > 0 && (b.BidAmount >= totalAmount)) {
				b.Amount = totalAmount
				totalAmount = 0
				b.Status = "success"
				successList  = append(successList, b)
			} else if (totalAmount > 0 && (b.BidAmount < totalAmount)) {
				b.Amount = b.BidAmount
				totalAmount -= b.Amount
				b.Status = "success"
				successList  = append(successList, b)
			} else if (totalAmount > 0) {
				test += " else"
			}
		}
	}

	newEnergy.SoldAmount = newEnergy.Amount - totalAmount
	if totalAmount == 0 {
		newEnergy.Status = "sold"
		returnMessage = "auction ended"
	} else {
		returnMessage = "auction continues"
	}

	if (newEnergy != energy) { 
		err = s.UpdateToken(ctx, newEnergy)
		if err != nil {
			return "", err
		}
		if (returnMessage == "auction continues") {
			returnMessage += " sametoken"
			returnMessage += test
		}
	} else {
		if (returnMessage == "auction continues") {
			returnMessage += " difftoken"
			returnMessage += test
		}
	}

	for _, bid := range successList {
		err = s.UpdateToken(ctx, bid)
		if err != nil {
			return "", err
		}
	}*/

	// return returnMessage, nil
}

func (s *SmartContract) AuctionEndQuery(ctx contractapi.TransactionContextInterface, id string, timestamp int64) ([]*Energy, error) {
	var energies []*Energy
	exist, err := s.EnergyExists(ctx, id)
	if (err != nil) {
		return energies, err
	}
	if exist == false {
		return energies, fmt.Errorf("no energy Token")
	}
	startTime := timestamp - auctionInterval * 75 * 1000000000
	endTime := timestamp - 1
	qEnergies, err := s.QueryAuctionEnd(ctx, id, startTime, endTime)
	if (err != nil) {
		return energies, err
	} 
	return qEnergies, nil
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


func (s *SmartContract) QueryAuctionEnd(ctx contractapi.TransactionContextInterface, energyId string, startTime int64, endTime int64) ([]*Energy, error) {
	queryString := fmt.Sprintf(`{"selector":{"DocType":"bid","Status":"bid","EnergyID":"%s","Bid Time":{"$gte":%d,"$lte":%d}},
	"use_index":["_design/indexAuctionEndDoc","indexAuctionEnd"]}`, energyId, startTime, endTime)
	// queryString := fmt.Sprintf(`{"selector":{"docType":"asset","owner":"%s"}}`, owner)

	energies, err := s.Query(ctx, queryString)

	return energies, err
}

func (s *SmartContract) QueryBid(ctx contractapi.TransactionContextInterface, status string, startTime int64, endTime int64) ([]*Energy, error) {
	queryString := fmt.Sprintf(`{"selector":{"DocType":"bid","Status":"%s","Bid Time":{"$gte":%d,"$lte":%d}},
	"use_index":["_design/indexBidResultDoc","indexBidResult"]}`, status, startTime, endTime)
	// queryString := fmt.Sprintf(`{"selector":{"docType":"asset","owner":"%s"}}`, owner)

	energies, err := s.Query(ctx, queryString)

	return energies, err
}

func (s *SmartContract) QueryByStatus(ctx contractapi.TransactionContextInterface, docType string, status string) ([]*Energy, error) {
	queryString := fmt.Sprintf(`{"selector":{"DocType":"%s","Status":"%s"},
	"use_index":["_design/indexStatusDoc","indexStatus"]}`, docType, status)
	// queryString := fmt.Sprintf(`{"selector":{"docType":"asset","owner":"%s"}}`, owner)

	energies, err := s.Query(ctx, queryString)

	return energies, err
}

func (s *SmartContract) QueryByTime(ctx contractapi.TransactionContextInterface, start int64, end int64) ([]*Energy, error) {
	queryString := fmt.Sprintf(`{"selector":{"DocType":"token","Generated Time":{"$gte":%d,"$lte":%d}},
	"use_index":["_design/indexTimeDoc","indexTime"]}`, start, end)
	// queryString := fmt.Sprintf(`{"selector":{"docType":"asset","owner":"%s"}}`, owner)

	energies, err := s.Query(ctx, queryString)

	return energies, err
}

func (s *SmartContract) BidOk2(ctx contractapi.TransactionContextInterface, energyId string, bidPrice float64, priority float64) (bool, error) {

	queryString := fmt.Sprintf(`{"selector":{"DocType":"bid","EnergyID":"%s","Bid Price":{"$gte":%v}},
	"use_index":["_design/indexBidOkDoc","indexBidOk"]}`, energyId, bidPrice)

	energy, err := s.ReadToken(ctx, energyId)
	if err != nil {
		return false, err
	}

	bidList, err := s.Query(ctx, queryString)
	if err != nil {
		return false, err
	}

	if (len(bidList) == 0) {
		return true, nil
	}

	var bidListTotalAmount float64 = 0
	for _, b := range bidList {
		if (b.BidPrice > bidPrice) {
			bidListTotalAmount += b.BidAmount
		} else if (b.BidPrice == bidPrice && b.Priority >= priority) {
			bidListTotalAmount += b.BidAmount
		}
		if (bidListTotalAmount >= energy.Amount - energy.SoldAmount) {
			return false, nil
		}
	}

	return true, nil

}

func (s *SmartContract) BidOk(ctx contractapi.TransactionContextInterface, energyId string, bidPrice float64, priority float64) (bool, error) {

	queryString := fmt.Sprintf(`{"selector":{"DocType":"bid","EnergyID":"%s","Bid Price":{"$gte":%v}},
	"use_index":["_design/indexBidOkDoc","indexBidOk"]}`, energyId, bidPrice)

	energy, err := s.ReadToken(ctx, energyId)
	if err != nil {
		return false, err
	}

	bidList, err := s.Query(ctx, queryString)
	if err != nil {
		return false, err
	}

	if (len(bidList) == 0) {
		return true, nil
	}

	var bidListTotalAmount float64 = 0
	for _, b := range bidList {
		bidListTotalAmount += b.BidAmount
		if (bidListTotalAmount >= energy.Amount - energy.SoldAmount) {
			return false, nil
		}
	}

	return true, nil

}
/*
func (s *SmartContract) QueryByUserAndBidTime(ctx contractapi.TransactionContextInterface, owner string, status string, startTime int64, endTime int64) ([]*Energy, error) {
	queryString := fmt.Sprintf(`{"selector":{"DocType":"token","Owner":"%s", "Status":"%s","Bid Time":{"$gte":%d,"$lte":%d}},
	"use_index":["_design/indexUserAndBidTimeDoc","indexUserAndBidTime"]}`, owner, status, startTime, endTime)

	energies, err := s.Query(ctx, queryString)

	return energies, err
}*/
/*
func (s *SmartContract) QueryByUserAndGeneratedTime(ctx contractapi.TransactionContextInterface, producer string, timestamp int64) ([]*Energy, error) {
	queryString := fmt.Sprintf(`{"selector":{"DocType":"token","Producer":"%s","Generated Time":%d},
	"use_index":["_design/indexUserAndGeneratedTimeDoc","indexUserAndGeneratedTime"]}`, producer, timestamp)
	
	energies, err := s.Query(ctx, queryString)

	return energies, err
}*/
/*
func (s *SmartContract) QueryByUserAndTime(ctx contractapi.TransactionContextInterface, producer string, startTime int64, endTime int64) ([]*Energy, error) {
	queryString := fmt.Sprintf(`{"selector":{"DocType":"token","Producer":"%s","Generated Time":{"$gte":%d,"$lte":%d}},
	"use_index":["_design/indexUserAndGeneratedTimeDoc","indexUserAndGeneratedTime"]}`, producer, startTime, endTime)

	energies, err := s.Query(ctx, queryString)

	return energies, err
}*/

/*
func (s *SmartContract) QueryByUserAndStatus(ctx contractapi.TransactionContextInterface, owner string, status string) ([]*Energy, error) {
	queryString := fmt.Sprintf(`{"selector":{"DocType":"token","Owner":"%s","Status":"%s"},
	"use_index":["_design/indexUserAndStatusDoc","indexUserAndStatus"]}`, owner, status)

	energies, err := s.Query(ctx, queryString)

	return energies, err
}*/

func (s *SmartContract) QueryByLocationRange(ctx contractapi.TransactionContextInterface,
	status string, owner string, latitudeLowerLimit float64, latitudeUpperLimit float64,
	longitudeLowerLimit float64, longitudeUpperLimit float64) ([]*Energy, error) {

	queryString := fmt.Sprintf(`{"selector":{"DocType":"token","Status":"%s", "Owner":{"$ne":"%s"},
	"Latitude":{"$gte":%f,"$lte":%f},"Longitude":{"$gte":%f,"$lte":%f}}, "use_index":["_design/indexLocationDoc","indexLocation"]}`,
		status, owner, latitudeLowerLimit, latitudeUpperLimit, longitudeLowerLimit, longitudeUpperLimit)

	energies, err := s.Query(ctx, queryString)

	return energies, err
}

func (s *SmartContract) Query(ctx contractapi.TransactionContextInterface, queryString string) ([]*Energy, error) {
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

func (s *SmartContract) DeleteAsset(ctx contractapi.TransactionContextInterface, id string) error {
	exists, err := s.EnergyExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the energy %s does not exist", id)
	}

	return ctx.GetStub().DelState(id)
}
