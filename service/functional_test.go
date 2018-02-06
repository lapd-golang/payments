package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func populateTestData(db *gorm.DB) (err error) {

	accounts := []Account{
		Account{Owner: "alice", Balance: 100.0, Currency: "USD"},
		Account{Owner: "bob", Balance: 10.0, Currency: "USD"},
		Account{Owner: "alice", Balance: 70.0, Currency: "PHP"},
		Account{Owner: "tmp", Balance: 1.0, Currency: "EUR"},
		Account{Owner: "tmp", Balance: 1.0, Currency: "EUR"},
		Account{Owner: "tmp", Balance: 1.0, Currency: "EUR"},
		Account{Owner: "tmp", Balance: 1.0, Currency: "EUR"},
		Account{Owner: "tmp", Balance: 1.0, Currency: "EUR"},
		Account{Owner: "tmp", Balance: 1.0, Currency: "EUR"},
		Account{Owner: "tmp", Balance: 1.0, Currency: "EUR"},
		Account{Owner: "tmp", Balance: 1.0, Currency: "EUR"},
		Account{Owner: "tmp", Balance: 1.0, Currency: "EUR"},
		Account{Owner: "tmp", Balance: 1.0, Currency: "EUR"},
		Account{Owner: "tmp", Balance: 1.0, Currency: "EUR"},
	}
	for _, acc := range accounts {
		if err = db.Create(&acc).Error; err != nil {
			return
		}
	}

	payments := []Payment{
		Payment{AccountID: 1, Amount: 1.0, Direction: "outgoing", AccountToID: 2},
		Payment{AccountID: 2, Amount: 1.0, Direction: "incoming", AccountFromID: 1},
		Payment{AccountID: 1, Amount: 1.0, Direction: "outgoing", AccountToID: 2},
		Payment{AccountID: 2, Amount: 1.0, Direction: "incoming", AccountFromID: 1},
		Payment{AccountID: 1, Amount: 1.0, Direction: "outgoing", AccountToID: 2},
		Payment{AccountID: 2, Amount: 1.0, Direction: "incoming", AccountFromID: 1},
		Payment{AccountID: 1, Amount: 1.0, Direction: "outgoing", AccountToID: 2},
		Payment{AccountID: 2, Amount: 1.0, Direction: "incoming", AccountFromID: 1},
		Payment{AccountID: 1, Amount: 1.0, Direction: "outgoing", AccountToID: 2},
		Payment{AccountID: 2, Amount: 1.0, Direction: "incoming", AccountFromID: 1},
		Payment{AccountID: 1, Amount: 1.0, Direction: "outgoing", AccountToID: 2},
		Payment{AccountID: 2, Amount: 1.0, Direction: "incoming", AccountFromID: 1},
		Payment{AccountID: 1, Amount: 1.0, Direction: "outgoing", AccountToID: 2},
		Payment{AccountID: 2, Amount: 1.0, Direction: "incoming", AccountFromID: 1},
	}
	for _, payment := range payments {
		if err = db.Create(&payment).Error; err != nil {
			return
		}
	}
	return
}

func functionalSetUp() (db *gorm.DB, engine *gin.Engine, err error) {
	gin.SetMode(gin.TestMode)

	if db, err = setupDatabase("sqlite3", "test.db"); err != nil {
		return
	}
	engine = setupRouter(db)
	if err = populateTestData(db); err != nil {
		return
	}
	return
}

func functionalTearDown(db *gorm.DB, engine *gin.Engine) {
	db.DropTableIfExists(&Account{})
	db.DropTableIfExists(&Payment{})
	db.Close()
}

func TestRealListAllAccounts(t *testing.T) {
	db, engine, err := functionalSetUp()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer functionalTearDown(db, engine)

	req, _ := http.NewRequest("GET", "/v1/accounts", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Response code should be %d, was: %d error: %s", http.StatusOK, w.Code, w.Body)
	}
	var respBody []Account
	if err := json.Unmarshal(w.Body.Bytes(), &respBody); err != nil {
		t.Error(err)
	}

	if len(respBody) != itemsPerPage {
		t.Errorf("Expected %d items, got %d", itemsPerPage, len(respBody))
	}

	item := respBody[1]
	if item.Balance != 10.0 || item.Currency != "USD" {
		t.Errorf("Wrong response, got %s", w.Body)
	}
}

func TestRealGetAllPayments(t *testing.T) {
	db, engine, err := functionalSetUp()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer functionalTearDown(db, engine)

	req, _ := http.NewRequest("GET", "/v1/payments", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Response code should be %d, was: %d (%s)", http.StatusOK, w.Code, w.Body)
	}

	var respBody []Payment
	if err := json.Unmarshal(w.Body.Bytes(), &respBody); err != nil {
		t.Error(err)
	}
	if len(respBody) != itemsPerPage {
		t.Errorf("Expected %d items, got %d", itemsPerPage, len(respBody))
	}
	payment := respBody[0]
	if payment.AccountID != 1 || payment.Amount != 1.0 || payment.AccountToID != 2 || payment.AccountFromID != 0 {
		t.Errorf("Wrong response, got %s", w.Body)
	}
}

func TestRealGetSingleAccountPayments(t *testing.T) {
	db, engine, err := functionalSetUp()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer functionalTearDown(db, engine)

	req, _ := http.NewRequest("GET", "/v1/payments?account_id=1", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Response code should be %d, was: %d (%s)", http.StatusOK, w.Code, w.Body)
	}

	var respBody []Payment
	if err := json.Unmarshal(w.Body.Bytes(), &respBody); err != nil {
		t.Error(err)
	}
	if len(respBody) != 7 {
		t.Errorf("Wrong response, got %s", w.Body)
	}
}

func TestRealSubmitSuccess(t *testing.T) {
	db, engine, err := functionalSetUp()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer functionalTearDown(db, engine)

	var dummy []Payment
	var beforeCount int
	if err := db.Find(&dummy).Count(&beforeCount).Error; err != nil {
		t.Error(err.Error())
	}

	req, _ := http.NewRequest("POST", "/v1/payments", bytes.NewBufferString(`{"from_account":1, "amount":50.0, "to_account":2}`))
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Response code should be %d, was: %d (%s)", http.StatusOK, w.Code, w.Body)
	}

	var afterCount int
	if err := db.Find(&dummy).Count(&afterCount).Error; err != nil {
		t.Error(err.Error())
	}

	// Two payments - incoming and outgoing
	if afterCount-beforeCount != 2 {
		t.Error("Wrong payments count")
	}

	testCases := []struct {
		id     uint
		amount float64
	}{
		{id: 1, amount: 50.0},
		{id: 2, amount: 60.0},
	}
	for _, testCase := range testCases {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/accounts?id=%d", testCase.id), nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Response code should be %d, was: %d", http.StatusOK, w.Code)
		}
		var respBody Account
		if err := json.Unmarshal(w.Body.Bytes(), &respBody); err != nil {
			t.Error(err)
		}

		if respBody.Balance != testCase.amount {
			t.Errorf("Wrong response, got %s", w.Body)
		}
	}

}

func TestRealSubmitError(t *testing.T) {
	db, engine, err := functionalSetUp()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer functionalTearDown(db, engine)

	testCases := []string{
		`{"from_account":1, "amount":500.0, "to_account":2}`, // Not enough balance
		`{"from_account":1, "amount":5.0, "to_account":1}`,   // Same destination
		`{"to_account":1, "amount":5.0, "from_account":100}`, // Wrong account
		`{"to_account":100, "amount":5.0, "from_account":1}`, // Wrong account
		`{"from_account":1, "amount":5.0, "to_account":3}`,   // Different currencies
		`{"to_account":1, "amount":5.0, "from_account":3}`,   // Different currencies
	}

	for _, testCase := range testCases {
		req, _ := http.NewRequest("POST", "/v1/payments", bytes.NewBufferString(testCase))
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Response code for %s should be %d, was: %d (%s)", testCase, http.StatusBadRequest, w.Code, w.Body)
		}
	}
}
