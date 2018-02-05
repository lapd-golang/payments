package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

const (
	itemsPerPage = 10
	defaultPage  = 0
)

func extractOffsetFromQuery(c *gin.Context) (uint64, error) {
	page, err := strconv.ParseUint(c.DefaultQuery("page", "0"), 10, 64)
	return page * itemsPerPage, err
}

func GetAccount(c *gin.Context, db *gorm.DB) {
	accountID, showSingleAccount := c.GetQuery("id")

	listAllAccounts := func() error {
		var offset uint64
		var err error
		if offset, err = extractOffsetFromQuery(c); err != nil {
			return err
		}

		var accounts []Account
		if err := db.Offset(offset).Limit(itemsPerPage).Find(&accounts).Error; err != nil {
			return err
		}
		c.JSON(http.StatusOK, accounts)
		return nil
	}

	listAccount := func() error {
		var res Account

		if err := db.First(&res, accountID).Error; err != nil {
			return err
			// return errors.New("No account found")
		}

		c.JSON(http.StatusOK, res)
		return nil
	}

	var actionFn func() error
	if showSingleAccount {
		actionFn = listAccount
	} else {
		actionFn = listAllAccounts
	}
	if err := actionFn(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func GetPayments(c *gin.Context, db *gorm.DB) {
	if err := func() error {
		var offset uint64
		var err error
		if offset, err = extractOffsetFromQuery(c); err != nil {
			return err
		}

		query := db
		accountID, filterByAccount := c.GetQuery("account_id")
		if filterByAccount {
			query = db.Where("account_id = ?", accountID)
		}

		var payments []Payment
		if err := query.Offset(offset).Limit(itemsPerPage).Find(&payments).Error; err != nil {
			return err
		}
		c.JSON(http.StatusOK, payments)
		return nil
	}(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func Submit(c *gin.Context, db *gorm.DB) {
	// The only write endpoint
	// Moreover, append only for payments and updates only (no insert/delete ops) for accounts
	var transfer Payment
	var sourceAccount, destAccount Account

	if err := func() error {
		if err := c.BindJSON(&transfer); err != nil {
			return err
		}
		if transfer.AccountFromID == 0 && transfer.AccountToID == 0 {
			return errors.New("Source or destination account is required")
		}
		if transfer.AccountFromID > 0 && transfer.AccountToID > 0 {
			return errors.New("Both source and destination accounts are specified")
		}
		// TODO: problem with dest_account -> wrong direction
		sourceAccount.ID, destAccount.ID = transfer.SourceDestinationID()
		if sourceAccount.ID == destAccount.ID {
			return errors.New("Source and destination accounts are the same")
		}
		return nil
	}(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	txn := db.Begin()
	if err := func() error {
		if err := db.First(&sourceAccount, sourceAccount.ID).Error; err != nil {
			return fmt.Errorf("No account with ID=%d", sourceAccount.ID)
		}
		if err := db.First(&destAccount, destAccount.ID).Error; err != nil {
			return fmt.Errorf("No account with ID=%d", sourceAccount.ID)
		}
		if sourceAccount.Currency != destAccount.Currency {
			return errors.New("Different currencies")
		}
		// Cheap balance check here
		if sourceAccount.Balance < transfer.Amount {
			return errors.New("Not enough balance")
		}

		sourceAccount.Balance -= transfer.Amount
		destAccount.Balance += transfer.Amount

		if err := db.Save(&sourceAccount).Error; err != nil {
			return err
		}
		if err := db.Save(&destAccount).Error; err != nil {
			return err
		}
		transfer.Direction = "outgoing"
		if err := db.Save(&transfer).Error; err != nil {
			return err
		}
		sndTransfer := transfer.InversePayment()
		sndTransfer.Direction = "incoming"
		if err := db.Save(&sndTransfer).Error; err != nil {
			return err
		}

		return nil
	}(); err != nil {
		txn.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	} else {
		if err := txn.Commit().Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusOK, gin.H{})
		}
	}
}
