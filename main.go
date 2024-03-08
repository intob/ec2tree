package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/intob/ec2tree/vm"
)

func main() {
	vmSvc, err := vm.NewSvc(&vm.Cfg{
		AMI:          "ami-01505b5fb77668db8",
		InstanceType: "t4g.nano",
		MaxDepth:     1,
		Fanout:       2,
		Region:       "eu-west-1",
	})
	if err != nil {
		panic(err)
	}

	root, err := vmSvc.CreateTree()
	if err != nil {
		panic(err)
	}

	// Set up channel to receive SIGINT
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt) // Register for SIGINT

	// Wait for SIGINT
	<-sigChan

	err = vmSvc.DeleteTree(root)
	if err != nil {
		panic(err)
	}

	fmt.Println("cleaned up tree")
}
