package main

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

type AnyTime struct{}

// Match satisfies sqlmock.Argument interface
func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

func setUp() (sqlmock.Sqlmock, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("can't create sqlmock: %s", err)
	}

	gormDB, gerr := gorm.Open("sqlite3", db)
	if gerr != nil {
		log.Fatalf("can't open gorm connection: %s", err)
	}
	gormDB.LogMode(true)

	return mock, gormDB.Set("gorm:update_column", true)
}

func tearDown(db *gorm.DB) {
	db.Close()
}

func TestListAllAccounts(t *testing.T) {
	sql, db := setUp()
	defer tearDown(db)
	engine := setupRouter(db)

	req, _ := http.NewRequest("GET", "/v1/accounts", nil)
	w := httptest.NewRecorder()
	columns := []string{"id"}
	sql.ExpectQuery(`SELECT \* FROM "accounts"`).
		WillReturnRows(sqlmock.NewRows(columns))

	engine.ServeHTTP(w, req)

	if err := sql.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	if w.Code != 200 {
		t.Errorf("Response code should be %d, was: %d", http.StatusOK, w.Code)
	}
}
func TestListAllAccountsWrongPage(t *testing.T) {
	_, db := setUp()
	defer tearDown(db)
	engine := setupRouter(db)

	req, _ := http.NewRequest("GET", "/v1/accounts?page=10x", nil)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Response code should be %d, was: %d", http.StatusBadRequest, w.Code)
	}
}
func TestListNonExistentAccount(t *testing.T) {
	sql, db := setUp()
	defer tearDown(db)
	engine := setupRouter(db)

	req, _ := http.NewRequest("GET", "/v1/accounts?id=10", nil)
	w := httptest.NewRecorder()
	sql.ExpectQuery(`SELECT \* FROM .+ "accounts"\."id"`).
		WithArgs("10").
		WillReturnError(fmt.Errorf("Some error"))
	engine.ServeHTTP(w, req)

	if err := sql.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Response code should be %d, was: %d", http.StatusBadRequest, w.Code)
	}
}
func TestListOneAccount(t *testing.T) {
	sql, db := setUp()
	defer tearDown(db)
	engine := setupRouter(db)

	req, _ := http.NewRequest("GET", "/v1/accounts?id=1", nil)
	w := httptest.NewRecorder()
	columns := []string{"id", "created_at", "updated_at", "deleted_at", "owner", "balance", "currency"}
	sql.ExpectQuery(`SELECT \* FROM .+ "accounts"\."id"`).
		WithArgs("1").
		WillReturnRows(sqlmock.NewRows(columns).AddRow(
			1, time.Now(), time.Now(), time.Now(), "alice", "155.0", "USD"))

	engine.ServeHTTP(w, req)

	if err := sql.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	if w.Code != 200 {
		t.Errorf("Response code should be %d, was: %d (%s)", http.StatusOK, w.Code, w.Body)
	}

	var respBody Account
	if err := json.Unmarshal(w.Body.Bytes(), &respBody); err != nil {
		t.Error(err)
	}
	if respBody.ID != 1 || respBody.Balance != 155.0 || respBody.Currency != "USD" {
		t.Errorf("Wrong response, got %s", w.Body)
	}
}
func TestGetAllPayments(t *testing.T) {
	sql, db := setUp()
	defer tearDown(db)
	engine := setupRouter(db)

	req, _ := http.NewRequest("GET", "/v1/payments", nil)
	w := httptest.NewRecorder()
	columns := []string{"id", "created_at", "updated_at", "deleted_at", "account_id", "amount", "direction", "account_to_id", "account_from_id"}
	sql.ExpectQuery(`SELECT \* FROM "payments"`).
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(1, time.Now(), time.Now(), time.Now(), 1, "155.0", "", 2, 0).
			AddRow(2, time.Now(), time.Now(), time.Now(), 1, "155.0", "", 0, 2).
			AddRow(3, time.Now(), time.Now(), time.Now(), 2, "155.0", "", 1, 0).
			AddRow(4, time.Now(), time.Now(), time.Now(), 2, "155.0", "", 1, 0).
			AddRow(5, time.Now(), time.Now(), time.Now(), 2, "155.0", "", 0, 1))

	engine.ServeHTTP(w, req)

	if err := sql.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	if w.Code != 200 {
		t.Errorf("Response code should be %d, was: %d (%s)", http.StatusOK, w.Code, w.Body)
	}

	var respBody []Payment
	if err := json.Unmarshal(w.Body.Bytes(), &respBody); err != nil {
		t.Error(err)
	}
	if len(respBody) != 5 {
		t.Errorf("Wrong response, got %s", w.Body)
	}
	payment := respBody[3]
	if payment.ID != 4 || payment.Amount != 155.0 || payment.AccountID != 2 || payment.AccountToID != 1 {
		t.Errorf("Wrong payment, got %s", payment)
	}
}
func TestGetSingleAccountPayments(t *testing.T) {
	sql, db := setUp()
	defer tearDown(db)
	engine := setupRouter(db)

	req, _ := http.NewRequest("GET", "/v1/payments?account_id=2", nil)
	w := httptest.NewRecorder()
	columns := []string{"id", "created_at", "updated_at", "deleted_at", "account_id", "amount", "direction", "account_to_id", "account_from_id"}
	sql.ExpectQuery(`SELECT \* FROM "payments"`).
		WithArgs("2").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(3, time.Now(), time.Now(), time.Now(), 2, "155.0", "", 1, 0).
			AddRow(4, time.Now(), time.Now(), time.Now(), 2, "155.0", "", 1, 0).
			AddRow(5, time.Now(), time.Now(), time.Now(), 2, "155.0", "", 0, 1))

	engine.ServeHTTP(w, req)

	if err := sql.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	if w.Code != 200 {
		t.Errorf("Response code should be %d, was: %d (%s)", http.StatusOK, w.Code, w.Body)
	}

	var respBody []Payment
	if err := json.Unmarshal(w.Body.Bytes(), &respBody); err != nil {
		t.Error(err)
	}
	if len(respBody) != 3 {
		t.Errorf("Wrong response, got %s", w.Body)
	}
}

func TestSubmitWrongRequest(t *testing.T) {
	_, db := setUp()
	defer tearDown(db)
	engine := setupRouter(db)

	testCases := []string{
		`{"account":1, "amount":50.0, "dest_account":1}`,
		`{"account":1, "amoun3t":50.0, "dest_account":1}`,
		`{"account":1, "amount":50.0, "src_account":2, "dest_account":2}`,
		`{"account":1, "amount":50.0, "src_account":1}`,
		`{"account":1, "amount":50.0}`,
		`{"account2":1, "amount":50.0}`,
	}

	for _, payload := range testCases {
		req, _ := http.NewRequest("POST", "/v1/payments", bytes.NewBufferString(payload))
		w := httptest.NewRecorder()

		engine.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Response code should be %d, was: %d (%s)", http.StatusBadRequest, w.Code, w.Body)
		}
	}
}
func TestSubmitSuccess(t *testing.T) {
	sql, db := setUp()
	defer tearDown(db)
	engine := setupRouter(db)

	req, _ := http.NewRequest("POST", "/v1/payments", bytes.NewBufferString(`{"account":1, "amount":50.0, "dest_account":2}`))
	w := httptest.NewRecorder()
	aColumns := []string{"id", "created_at", "updated_at", "deleted_at", "owner", "balance", "currency"}
	// pColumns := []string{"id", "created_at", "updated_at", "deleted_at", "account_id", "amount", "direction", "account_to_id", "account_from_id"}

	sql.ExpectBegin()
	sql.ExpectQuery(`SELECT \* FROM .+ WHERE .+ "accounts"\."id" =`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows(aColumns).
			AddRow(1, time.Time{}, time.Time{}, nil, "alice", 155.0, "USD"))
	sql.ExpectQuery(`SELECT \* FROM .+ "accounts"\."id" =`).
		WithArgs(2, 2).
		WillReturnRows(sqlmock.NewRows(aColumns).
			AddRow(2, time.Time{}, time.Time{}, nil, "bob", 5.0, "USD"))
	sql.ExpectExec(`UPDATE "accounts" SET`).
		WithArgs(time.Time{}, time.Time{}, nil, "alice", 105.0, "USD", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	sql.ExpectExec(`UPDATE "accounts" SET`).
		WithArgs(time.Time{}, time.Time{}, nil, "bob", 55.0, "USD", 2).
		WillReturnResult(sqlmock.NewResult(0, 1))
	sql.ExpectExec(`INSERT INTO "payments"`).
		WithArgs(AnyTime{}, AnyTime{}, nil, 1, 50.0, "outgoing", 2, 0).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sql.ExpectExec(`INSERT INTO "payments"`).
		WithArgs(AnyTime{}, AnyTime{}, nil, 2, 50.0, "incoming", 0, 1).
		WillReturnResult(sqlmock.NewResult(2, 1))
	sql.ExpectCommit()

	engine.ServeHTTP(w, req)

	if err := sql.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
	if w.Code != http.StatusOK {
		t.Errorf("Response code should be %d, was: %d (%s)", http.StatusOK, w.Code, w.Body)
	}
}

func TestSubmitCommitFailure(t *testing.T) {
	sql, db := setUp()
	defer tearDown(db)
	engine := setupRouter(db)

	req, _ := http.NewRequest("POST", "/v1/payments", bytes.NewBufferString(`{"account":1, "amount":50.0, "dest_account":2}`))
	w := httptest.NewRecorder()
	aColumns := []string{"id", "created_at", "updated_at", "deleted_at", "owner", "balance", "currency"}
	// pColumns := []string{"id", "created_at", "updated_at", "deleted_at", "account_id", "amount", "direction", "account_to_id", "account_from_id"}

	sql.ExpectBegin()
	sql.ExpectQuery(`SELECT \* FROM .+ WHERE .+ "accounts"\."id" =`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows(aColumns).
			AddRow(1, time.Time{}, time.Time{}, nil, "alice", 155.0, "USD"))
	sql.ExpectQuery(`SELECT \* FROM .+ "accounts"\."id" =`).
		WithArgs(2, 2).
		WillReturnRows(sqlmock.NewRows(aColumns).
			AddRow(2, time.Time{}, time.Time{}, nil, "bob", 5.0, "USD"))
	sql.ExpectExec(`UPDATE "accounts" SET`).
		WithArgs(time.Time{}, time.Time{}, nil, "alice", 105.0, "USD", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	sql.ExpectExec(`UPDATE "accounts" SET`).
		WithArgs(time.Time{}, time.Time{}, nil, "bob", 55.0, "USD", 2).
		WillReturnResult(sqlmock.NewResult(0, 1))
	sql.ExpectExec(`INSERT INTO "payments"`).
		WithArgs(AnyTime{}, AnyTime{}, nil, 1, 50.0, "outgoing", 2, 0).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sql.ExpectExec(`INSERT INTO "payments"`).
		WithArgs(AnyTime{}, AnyTime{}, nil, 2, 50.0, "incoming", 0, 1).
		WillReturnResult(sqlmock.NewResult(2, 1))
	sql.ExpectCommit().
		WillReturnError(errors.New("Error 4025: CONSTRAINT `positive_balance` failed for `test`.`accounts`"))

	engine.ServeHTTP(w, req)

	if err := sql.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Response code should be %d, was: %d (%s)", http.StatusBadRequest, w.Code, w.Body)
	}
}

func TestSubmitError(t *testing.T) {
	sql, db := setUp()
	defer tearDown(db)
	engine := setupRouter(db)

	req, _ := http.NewRequest("POST", "/v1/payments", bytes.NewBufferString(`{"account":1, "amount":50.0, "dest_account":2}`))
	w := httptest.NewRecorder()
	aColumns := []string{"id", "created_at", "updated_at", "deleted_at", "owner", "balance", "currency"}
	// pColumns := []string{"id", "created_at", "updated_at", "deleted_at", "account_id", "amount", "direction", "account_to_id", "account_from_id"}

	sql.ExpectBegin()
	sql.ExpectQuery(`SELECT \* FROM .+ WHERE .+ "accounts"\."id" =`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows(aColumns).
			AddRow(1, time.Time{}, time.Time{}, nil, "alice", 155.0, "USD"))
	sql.ExpectQuery(`SELECT \* FROM .+ "accounts"\."id" =`).
		WithArgs(2, 2).
		WillReturnRows(sqlmock.NewRows(aColumns).
			AddRow(2, time.Time{}, time.Time{}, nil, "bob", 5.0, "EUR"))
	sql.ExpectRollback()

	engine.ServeHTTP(w, req)

	if err := sql.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Response code should be %d, was: %d (%s)", http.StatusBadRequest, w.Code, w.Body)
	}
}
