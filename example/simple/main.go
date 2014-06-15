package main

import (
	"fmt"
)

func main() {
	fmt.Println("Start")
	func1()
	a := 42
	fmt.Println("one", a)
	a += func2()
	fmt.Println("two", a)
	fmt.Println("three")
}

func func1() {
	fmt.Println("In func1")
}

func func2() int {
	b := 12
	b *= b
	return b
}
