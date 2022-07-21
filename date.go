package main

import (
	"fmt"
	"time"
)

func date() string {
	ddmmyyyy := time.Now().Format("02/01/2006")
	mmddyyyy := time.Now().Format("01/02/2006")
	yyyymmdd := time.Now().Format("2006/01/02")
	unixepoch := time.Now().Unix()

	return fmt.Sprintf("ddmmyyyy\n%s\nmmddyyyy\n%s\nyyyymmdd\n%s\nunix epoch\n%d", ddmmyyyy, mmddyyyy, yyyymmdd, unixepoch)
}
