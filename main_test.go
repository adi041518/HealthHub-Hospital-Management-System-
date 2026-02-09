package main

import (
	"testing"

	server "github.com/KanapuramVaishnavi/Core/server"
	"github.com/gin-gonic/gin"
)

func TestRun_FullCoverage(t *testing.T) {
	isTest = true
	defer func() { isTest = false }()

	var capturedOpts server.Options

	// intercept options
	startServer = func(opts server.Options) {
		capturedOpts = opts
	}

	// run main logic
	main()
	run()

	// 🔥 EXECUTE ALL HANDLERS (this is the missing part)
	capturedOpts.JobsHandler()
	capturedOpts.MigrationHandler()
	capturedOpts.WebServerPreHandler(gin.New())
}
