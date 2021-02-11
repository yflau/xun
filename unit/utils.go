package unit

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/fatih/color"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/logger"
)

// "root:123456@tcp(192.168.31.119:3306)/xiang?charset=utf8mb4&parseTime=True&loc=Local"

// DSN the dsns for testing
var dsns map[string]string = map[string]string{
	"mysql":   os.Getenv("XUN_UNIT_MYSQL_DSN"),
	"sqlite3": os.Getenv("XUN_UNIT_SQLITE3_DSN"),
	"pgsql":   os.Getenv("XUN_UNIT_POSTGRE_DSN"),
	"oracle":  os.Getenv("XUN_UNIT_ORACLE_DSN"),
	"sqlsvr":  os.Getenv("XUN_UNIT_SQLSVR_DSN"),
}

// Use create a capsule intance using DSN
func Use(driver string) *capsule.Manager {
	dsn := DSN(driver)
	return capsule.AddConn("primary", driver, dsn)
}

// DSN get the dsn from evn
func DSN(name string) string {
	dsn, has := dsns[name]
	if !has || dsn == "" {
		err := errors.New("dsn not found!" + name)
		panic(err)
	}
	return dsn
}

// SetLogger set the unit file logger
func SetLogger() {
	logfile := os.Getenv("XUN_UNIT_LOG")
	output := os.Stdout
	if logfile != "" {
		f, err := os.OpenFile(logfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err == nil {
			output = f
		}
	}
	logger.DefaultLogger.SetOutput(output)
	logger.DefaultErrorLogger.SetOutput(output)
}

// Catch and out
func Catch() {
	if r := recover(); r != nil {
		switch r.(type) {
		case string:
			color.Red("%s\n", r)
			break
		case error:
			color.Red("%s\n", r.(error).Error())
			break
		default:
			color.Red("%#v\n", r)
		}
		fmt.Println(string(debug.Stack()))
	}
}
