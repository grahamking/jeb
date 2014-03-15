package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"strings"
)

func main() {
	l, err := net.Listen("unix", "/tmp/jeb-socket")
	if err != nil {
		log.Fatal(err)
	}
	c, err := l.Accept()
	if err != nil {
		log.Fatal(err)
	}
	for {
		buf := make([]byte, 1024)
		numRead, err := c.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		parts := strings.Split(string(buf[:numRead]), ":")
		if len(parts) != 3 {
			log.Fatalf("Expected 3 parts, got %v", parts)
		}

		filename := parts[0]
		line, err := strconv.Atoi(parts[1])
		if err != nil {
			log.Fatal(err)
		}
		//function := parts[2]
		display(filename, line)
	}
	c.Close()
	l.Close()
}

func display(filename string, lineNum int) {
	filedata, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(filedata), "\n")
	for i, line := range lines {
		if i+1 == lineNum {
			fmt.Println(bold(line))
		} else {
			fmt.Println(line)
		}
	}
	fmt.Println("-----")
}

func bold(str string) string {
	return "\033[1m" + str + "\033[0m"
}
