package main

import (
	"os"

	"github.com/DwifteJB/aplsonic/src/cmd/createAccount"
	resetadmin "github.com/DwifteJB/aplsonic/src/cmd/reset-admin"
	"github.com/DwifteJB/aplsonic/src/serve"
)

func main() {
	// get cmd args
	args := os.Args

	if len(args) < 2 {
		println("Please provide a command. Available commands: create-account, serve")
		return
	}

	// prob a more elegant way for this lol
	switch args[1] {
	case "create-account":
		createAccount.CMD(args[2:])
	case "serve":
		serve.Serve()
	case "reset-admin":
		resetadmin.CMD()
	default:
		println("Unknown command. Available commands: create-account, serve, reset-admin")
	}
}
