package main

import (
	"errors"
	"fmt"

	"github.com/jinzhu/gorm"
)

// Account type represent physical bank account with "should-always-stay-positive"
// balance field, owner and currency fields. Assuming only transactions between
// accounts with the same currencies are allowed.
type Account struct {
	gorm.Model

	Owner    string
	Balance  float64
	Currency string
}

// Payment (or transfer) describe balance (money) transfer between accounts.
// API allows to specify source and destination.
// AccountID specifies what account this transfer applies to, Direction specifies
// is it either `incoming` transfer or `outgoing`.
// There always should be reciprocal transfer for other account involved: that is,
// identified by either AccountTo or AccountFrom IDs.
// Amount number should always be positive.
type Payment struct {
	gorm.Model

	AccountID     uint    `json:"account"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	Direction     string
	AccountToID   uint `json:"to_account" binding:"required"`
	AccountFromID uint `json:"from_account" binding:"required"`
}

// Transfer applies payment to tow involved accounts.
// Checks for same currency and that source account has enough balance
// Returns error if transfer is not possible, nil otherwise.
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

func (p Payment) Outgoing() (res Payment) {
	res.AccountID = p.AccountFromID
	res.AccountToID = p.AccountToID
	res.Direction = "outgoing"
	res.Amount = p.Amount
	return res
}

func (p Payment) Incoming() (res Payment) {
	res.AccountID = p.AccountToID
	res.AccountFromID = p.AccountFromID
	res.Direction = "incoming"
	res.Amount = p.Amount
	return res
}

func (p Payment) String() string {
	return fmt.Sprintf("ID=%d, FROM=%d, TO=%d, Amount=%f",
		p.AccountID, p.AccountFromID, p.AccountToID, p.Amount)
}
