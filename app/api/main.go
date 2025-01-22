package main

import (
	"Gocument/app/api/internal/flag"
	"Gocument/app/api/internal/initialize"
	"Gocument/app/api/router"
)

func main() {
	initialize.SetupViper()
	initialize.SetupLogger()
	initialize.SetupDatabase()
	initialize.SetUpCos()

	option := flag.Parse()
	if flag.IsWebStop(option) {
		flag.SwitchOption(option)
	}

	router.InitRouter()
}
