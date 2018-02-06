package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Constant for pagination
const (
	itemsPerPage = 10
	defaultPage  = 0
)

// extractOffsetFromQuery extracts offset and count from query parameters
// It waits for page argument and translate it into offset/limit
func extractOffsetFromQuery(c *gin.Context) (uint64, error) {
	page, err := strconv.ParseUint(c.DefaultQuery("page", "0"), 10, 64)
	return page * itemsPerPage, err
}

// getObjects is a helper function that gets a list of object from a database
// and writes them as JSON into http response. Allows for pagination (see extractOffsetFromQuery())
// Returns nil on success and error otherwise
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

// GetAccount is a handler for /account endpoint.
// It lists all account by default or list only one if `id` query parameter is present
// in a query string.
// Writes results in JSON format.
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

// GetPayments is a handler for /payments endpoint.
// It lists all payments by default or only those related to specified in a
// querty strin `account_id`.
// Writes results in JSON format.
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

// validatePaymentPayoload validates payload for /payment POST endpoint.
// See `Payment`` struct for details. Also checks source and destination IDs.
// Returns nil on success and error otherwise.
func validatePaymentPayload(c *gin.Context, payment *Payment) error {
	if err := c.BindJSON(payment); err != nil {
		return err
	}
	if payment.AccountFromID == payment.AccountToID {
		return errors.New("Source and destination accounts are the same")
	}
	return nil
}

// saveObjects is helper function to write objects to a database
// Returns nil on success, error otherwise
func saveObjects(db *gorm.DB, objs []interface{}) error {
	for _, obj := range objs {
		if err := db.Save(obj).Error; err != nil {
			return err
		}
	}
	return nil
}

// Submit is a handler for POST /payment endpoint.
// It's the only write endpoint. Database transaction is used to guarantee integrity.
// For non-sqlite database engines it uses database `check` constraint to ensure
// positive balance (second check for concurrent transactions).
func Submit(c *gin.Context, db *gorm.DB) {
	var payment Payment
	var sourceAccount, destAccount Account

	if err := validatePaymentPayload(c, &payment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	txn := db.Begin()
	if err := func() error {
		sourceID, destID := payment.AccountFromID, payment.AccountToID
		if err := db.First(&sourceAccount, sourceID).Error; err != nil {
			return fmt.Errorf("No account with ID=%d", sourceID)
		}
		if err := db.First(&destAccount, destID).Error; err != nil {
			return fmt.Errorf("No account with ID=%d", destID)
		}

		if err := payment.Transfer(&sourceAccount, &destAccount); err != nil {
			return err
		}
		fromPayment, toPayment := payment.Outgoing(), payment.Incoming()

		if err := saveObjects(db, []interface{}{
			&sourceAccount,
			&destAccount,
			&fromPayment,
			&toPayment,
		}); err != nil {
			return err
		}

		return nil
	}(); err != nil {
		txn.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	} else {
		// We still can fail here: transaction can fail even if previous
		// programmatic balance check succeeds.
		if err := txn.Commit().Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusOK, gin.H{})
		}
	}
}
