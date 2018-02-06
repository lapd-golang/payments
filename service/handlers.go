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

func getObjects(c *gin.Context, db *gorm.DB, out interface{}) error {
	var offset uint64
	var err error
	if offset, err = extractOffsetFromQuery(c); err != nil {
		return err
	}

	if err := db.Offset(offset).Limit(itemsPerPage).Find(out).Error; err != nil {
		return err
	}
	c.JSON(http.StatusOK, out)
	return nil
}

func GetAccount(c *gin.Context, db *gorm.DB) {
	accountID, showSingleAccount := c.GetQuery("id")

	listAllAccounts := func() error {
		var accounts []Account
		return getObjects(c, db, &accounts)
	}

	listAccount := func() error {
		var res Account
		if err := db.First(&res, accountID).Error; err != nil {
			return err
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
	accountID, filterByAccount := c.GetQuery("account_id")
	query := db
	if filterByAccount {
		query = db.Where("account_id = ?", accountID)
	}

	var payments []Payment
	if err := getObjects(c, query, &payments); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func validatePaymentPayload(c *gin.Context, payment *Payment) error {
	if err := c.BindJSON(payment); err != nil {
		return err
	}
	if payment.AccountFromID == 0 && payment.AccountToID == 0 {
		return errors.New("Source or destination account is required")
	}
	if payment.AccountFromID > 0 && payment.AccountToID > 0 {
		return errors.New("Both source and destination accounts are specified")
	}
	sourceID, destID := payment.SourceID(), payment.DestinationID()
	if sourceID == destID {
		return errors.New("Source and destination accounts are the same")
	}
	return nil
}

func saveObjects(db *gorm.DB, objs []interface{}) error {
	for _, obj := range objs {
		if err := db.Save(obj).Error; err != nil {
			return err
		}
	}
	return nil
}

func Submit(c *gin.Context, db *gorm.DB) {
	// The only write endpoint
	// Moreover, append only for payments and updates only (no insert/delete ops) for accounts
	var payment Payment
	var sourceAccount, destAccount Account

	if err := validatePaymentPayload(c, &payment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	txn := db.Begin()
	if err := func() error {
		sourceID, destID := payment.SourceID(), payment.DestinationID()
		if err := db.First(&sourceAccount, sourceID).Error; err != nil {
			return fmt.Errorf("No account with ID=%d", sourceID)
		}
		if err := db.First(&destAccount, destID).Error; err != nil {
			return fmt.Errorf("No account with ID=%d", destID)
		}

		payment.Direction = "outgoing"
		if err := payment.Transfer(&sourceAccount, &destAccount); err != nil {
			return err
		}
		sndPayment := payment.InversePayment()
		sndPayment.Direction = "incoming"

		if err := saveObjects(db, []interface{}{
			&sourceAccount,
			&destAccount,
			&payment,
			&sndPayment,
		}); err != nil {
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
