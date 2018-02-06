# Overview

Simple RESTful service for account and payments management written in Golang with [GORM](http://gorm.io/database.html) as ORM component and [Gin](https://gin-gonic.github.io/gin/) as web framework and HTTP router.

# API

Endpoints:

 - GET `v1/accounts` lists all accounts. `page` and `id` are recognized as query parameters
 - GET `v1/payments` lists all payments. `page` and `account_id` are recognized as query parameters
 - POST `v1/payments` submit a payment. Expects `application/json` payload with `from_account`, `to_account` and `amount` fields.

## Installation

Installation is as simple as:

```
$ go get -u github.com/rampage644/payments
$ go get -u github.com/golang/dep/cmd/dep
$ cd $GOPATH/src/github.com/rampage644/payments
$ dep ensure
$ cd -

$ go install github.com/rampage644/payments/service
```

# Usage

Application could be run with:

```
$ $GOPATH/bin/service --dialect mysql --connect 'connection_string'
```

It accepts `--connect` and `--dialect` switches to specify dialect (database) and connection string (database specific). See more at <http://gorm.io/database.html#connecting-to-a-database>.



## Development

Get sources with:

```
go get -u github.com/rampage644/payments
```

Run tests with coverage:

```
cd $GOPATH/src/github.com/rampage644/payments/service
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Playground

Start MySQL database, I prefer to use Docker for such purposes:

```
$ docker run  --name mariadb-server  -e MYSQL_ROOT_PASSWORD=secret -e MYSQL_DATABASE=test -d mariadb
```

Find running instance IP address:

```
$ docker inspect mariadb-server | grep IPAddress
```

Start service with `--connect` string (replace IP address with actual one):

```
$ $GOPATH/bin/service --connect 'root:secret@(172.17.0.2:3306)/test?charset=utf8&parseTime=True&loc=Local'
```

Another option is to run it in Docker container. First build the image:

```
$ cd $GOPATH/src/github.com/rampage644/payments/service
$ docker build . -t service:latest
```

Then, run service with the connect string:

```
$ docker run service:latest --name service /go/bin/service --connect 'root:secret@(172.17.0.2:3306)/test?charset=utf8&parseTime=True&loc=Local'
```

Insert some test data into database:

```
$ docker run -it --link mariadb-server --rm mariadb sh -c 'exec mysql -h"172.17.0.2"  -uroot -p"secret"'
MariaDB [(none)]> use test;
Database changed

MariaDB [test]> insert into accounts (owner, balance, currency) values ("alice", 100.0, "USD");
Query OK, 1 row affected (0.01 sec)

MariaDB [test]> insert into accounts (owner, balance, currency) values ("bob", 200.0, "USD");
Query OK, 1 row affected (0.01 sec)

MariaDB [test]> insert into accounts (owner, balance, currency) values ("zhao", 10.0, "EUR");
Query OK, 1 row affected (0.01 sec)

```

Test its API. If run inside a docker container, get container IP address first:

```
$ docker inspect service | grep IPAddress
```

And use it instead of `localhost`.

```
$ pip install httpie

$ http GET localhost:8080/v1/accounts
HTTP/1.1 200 OK
Content-Length: 426
Content-Type: application/json; charset=utf-8
Date: Tue, 06 Feb 2018 10:49:41 GMT

[
    {
        "Balance": 100,
        "CreatedAt": "0001-01-01T00:00:00Z",
        "Currency": "USD",
        "DeletedAt": null,
        "ID": 1,
        "Owner": "alice",
        "UpdatedAt": "0001-01-01T00:00:00Z"
    },
    {
        "Balance": 200,
        "CreatedAt": "0001-01-01T00:00:00Z",
        "Currency": "USD",
        "DeletedAt": null,
        "ID": 2,
        "Owner": "bob",
        "UpdatedAt": "0001-01-01T00:00:00Z"
    },
    {
        "Balance": 10,
        "CreatedAt": "0001-01-01T00:00:00Z",
        "Currency": "EUR",
        "DeletedAt": null,
        "ID": 3,
        "Owner": "zhao",
        "UpdatedAt": "0001-01-01T00:00:00Z"
    }
]


$ http POST localhost:8080/v1/payments from_account:=1 to_account:=2 amount:=10
HTTP/1.1 200 OK
Content-Length: 2
Content-Type: application/json; charset=utf-8
Date: Tue, 06 Feb 2018 10:51:10 GMT

{}


$ http GET 'localhost:8080/v1/accounts?id=1'
HTTP/1.1 200 OK
Content-Length: 146
Content-Type: application/json; charset=utf-8
Date: Tue, 06 Feb 2018 10:51:31 GMT

{
    "Balance": 90,
    "CreatedAt": "0001-01-01T00:00:00Z",
    "Currency": "USD",
    "DeletedAt": null,
    "ID": 1,
    "Owner": "alice",
    "UpdatedAt": "2018-02-06T13:51:10+03:00"
}


$ http GET 'localhost:8080/v1/payments?account_id=2'
HTTP/1.1 200 OK
Content-Length: 186
Content-Type: application/json; charset=utf-8
Date: Tue, 06 Feb 2018 10:52:46 GMT

[
    {
        "CreatedAt": "2018-02-06T13:51:10+03:00",
        "DeletedAt": null,
        "Direction": "incoming",
        "ID": 2,
        "UpdatedAt": "2018-02-06T13:51:10+03:00",
        "account": 2,
        "amount": 10,
        "from_account": 1,
        "to_account": 0
    }
]
```
