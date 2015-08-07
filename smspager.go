package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("smspager: device routes.json")
		os.Exit(1)
	}
	routes, err := ioutil.ReadFile(os.Args[2])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	sc, err := New(os.Args[1], 115200, routes)
	if err != nil {
		fmt.Println(fmt.Sprintf("ListSMS: %s", err))
		os.Exit(1)
	}
	sc.ForwardSMS()
	os.Exit(0)
}
