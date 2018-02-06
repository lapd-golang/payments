package main

import (
	"flag"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

// setupDatabase opens database "connection" (connection pool to be more
// strict) and migrates schema
func setupDatabase(dialect string, connect string) (*gorm.DB, error) {
	log.Printf("Using %s dialect, connection string is %s", dialect, connect)
	db, err := gorm.Open(dialect, connect)
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&Account{})
	db.AutoMigrate(&Payment{})

	// As `gorm` doesn't have constraints we have to do this manually,
	// there is open PR for that.
	// This will not work with sqlite3 as it expects CONSTRAINTs to be defined
	// during CREATE TABLE statement.
	if err := db.Raw(`ALTER TABLE accounts ADD CONSTRAINT positive_balance CHECK (balance >= 0);`).Error; err != nil {
		log.Println(err.Error())
	}
	return db, nil
}

// setupRouter will create GIN router engine fot http request and provide
// handlers with a "database connection pool".
func setupRouter(db *gorm.DB) *gin.Engine {
	router := gin.Default()

	v1 := router.Group("/v1")
	v1.GET("/accounts", func(c *gin.Context) {
		GetAccount(c, db)
	})
	v1.GET("/payments", func(c *gin.Context) {
		GetPayments(c, db)
	})
	v1.POST("/payments", func(c *gin.Context) {
		Submit(c, db)
	})

	return router
}

func main() {
	// dialect
	dialect := flag.String("dialect", "mysql", "Database to use; see gorm dialects")
	connect := flag.String("connect", "root:secret@/test?charset=utf8&parseTime=True&loc=Local", "DSN connection string")
	flag.Parse()

	db, err := setupDatabase(*dialect, *connect)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()

	router := setupRouter(db)
	router.Run()
}
