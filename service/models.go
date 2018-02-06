package main

import (
	"errors"

	"github.com/jinzhu/gorm"
)

type Account struct {
	gorm.Model

	Owner    string
	Balance  float64
	Currency string
}

type Payment struct {
	gorm.Model

	AccountID     uint    `json:"account" binding:"required"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	Direction     string
	AccountToID   uint `json:"dest_account"`
	AccountFromID uint `json:"src_account"`
}

func (p *Payment) Transfer(source *Account, dest *Account) error {
	if source.Currency != dest.Currency {
		return errors.New("Different currencies")
	}
	// Cheap balance check here
	if source.Balance < p.Amount {
		return errors.New("Not enough balance")
	}

	source.Balance -= p.Amount
	dest.Balance += p.Amount
	return nil
}

func (p *Payment) SourceID() uint {
	if p.AccountFromID > 0 {
		return p.AccountFromID
	}
	if p.AccountToID > 0 {
		return p.AccountID
	}
	return 0
}

func (p *Payment) DestinationID() uint {
	if p.AccountFromID > 0 {
		return p.AccountID
	}
	if p.AccountToID > 0 {
		return p.AccountToID
	}
	return 0
}
func (p Payment) InversePayment() (res Payment) {
	if p.AccountFromID > 0 {
		res.AccountID = p.AccountFromID
		res.AccountToID = p.AccountID
	}
	if p.AccountToID > 0 {
		res.AccountID = p.AccountToID
		res.AccountFromID = p.AccountID
	}
	res.Amount = p.Amount
	return res
}

// func (p Payment) String() string {
// 	return fmt.Sprintf("ID=%d, FROM=%d, TO=%d, Amount=%f",
// 		p.AccountID, p.AccountFromID, p.AccountToID, p.Amount)
// }
