package main

import (
	"fmt"
	"jeb/client"
)

func main() {
	client.Trace("example/simple.go", "8", "main")
	fmt.Println("Start")
	client.Trace("example/simple.go", "9", "main")
	a := 42
	client.Trace("example/simple.go", "10", "main")
	fmt.Println("one", a)
	client.Trace("example/simple.go", "11", "main")
	a++
	client.Trace("example/simple.go", "12", "main")
	fmt.Println("two", a)
	client.Trace("example/simple.go", "13", "main")
	fmt.Println("three")
}
