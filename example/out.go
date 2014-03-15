package main

import (
	"fmt"
	"jeb/client"
)

func main() {
	client.Trace("example/simple.go", "8", "main")
	fmt.Println("Start")
	client.Trace("example/simple.go", "9", "main")
	func1()
	client.Trace("example/simple.go", "10", "main")
	a := 42
	client.Trace("example/simple.go", "11", "main")
	fmt.Println("one", a)
	client.Trace("example/simple.go", "12", "main")
	a += func2()
	client.Trace("example/simple.go", "13", "main")
	fmt.Println("two", a)
	client.Trace("example/simple.go", "14", "main")
	fmt.Println("three")
}

func func1() {
	client.Trace("example/simple.go", "18", "func1")
	fmt.Println("In func1")
}

func func2() int {
	client.Trace("example/simple.go", "22", "func2")
	b := 12
	client.Trace("example/simple.go", "23", "func2")
	b *= b
	client.Trace("example/simple.go", "24", "func2")
	return b
}
