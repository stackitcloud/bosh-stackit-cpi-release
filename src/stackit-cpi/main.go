package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/stackitcloud/stackit-cpi/cpi"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

var configPathOpt = flag.String("configPath", "", "Path to configuration file")

func main() {
	flag.Parse()
	if *configPathOpt == "" {
		err := fmt.Errorf("no -configPath received in cpi call %s", os.Args)
		fmt.Fprintf(os.Stdout, "%s", lib.ConvertErrToRPCResponse(err))
		os.Exit(1)
	}

	req, err := cpi.ParseStdIn()
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s", lib.ConvertErrToRPCResponse(err))
		os.Exit(1)
	}

	c, err := cpi.New(*configPathOpt)
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s", lib.ConvertErrToRPCResponse(err))
		os.Exit(1)
	}
	logCtx := req.GetLoggingContext()
	c.Log.WithContext(logCtx)

	resp, err := c.Dispatch(req)
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s", resp)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "%s", resp)
	os.Exit(0)
}
