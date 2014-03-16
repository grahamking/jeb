package main

import (
	"github.com/nsf/termbox-go"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"strings"
)

func runServer() {
	termbox.Init()
	defer termbox.Close()

	l, err := net.Listen("unix", "/tmp/jeb-socket")
	if err != nil {
		log.Fatal(err)
	}

	out(0, 0, "Waiting for message from client...")
	termbox.Flush()

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
		waitForInput()
		c.Write([]byte{'\n'})
	}
	c.Close()
	l.Close()
}

func waitForInput() {
loop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			break loop
		default:
			continue
		}
	}
}

func display(filename string, lineNum int) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	filedata, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(filedata), "\n")
	for i, line := range lines {
		if i+1 == lineNum {
			tbprint(0, i, termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault, line)
		} else {
			out(0, i, line)
		}
	}
	termbox.Flush()
}

func bold(str string) string {
	return "\033[1m" + str + "\033[0m"
}

func out(x, y int, msg string) {
	tbprint(x, y, termbox.ColorDefault, termbox.ColorDefault, msg)
}
func tbprint(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x++
	}
}
