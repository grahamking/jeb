package main

import (
	"errors"
	"github.com/nsf/termbox-go"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

func runServer() {
	termbox.Init()
	defer termbox.Close()

	setupLogging()

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

	log.Println("NewJeb")
	j := NewJeb(c)
	log.Println("j.run")
	j.run()

	l.Close()
}

func setupLogging() {
	l, err := os.Create("jeb.log")
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(l)
}

type Jeb struct {
	c              net.Conn
	buf            []byte
	files          map[string][]string
	skipInFunction string
}

func NewJeb(c net.Conn) *Jeb {
	return &Jeb{
		c:     c,
		buf:   make([]byte, 1024),
		files: make(map[string][]string),
	}
}

func (j *Jeb) run() {
	for {
		log.Println("receive")
		numRead := j.receive()
		if numRead == 0 {
			break
		}

		log.Println("parse")
		filename, line, function := j.parse(numRead)

		if j.skipInFunction == "" || j.skipInFunction == function {
			log.Println("display: ", filename, line)
			j.display(filename, line)

			input, err := j.waitForInput()
			log.Println("input: ", input)
			if err != nil {
				break
			}
			if input == 's' {
				// step in
				j.skipInFunction = ""
			} else if input == 'n' {
				// next (step over)
				j.skipInFunction = function
			}

		}
		log.Println("proceed")
		j.proceed()
	}
	j.c.Close()
}

func (j *Jeb) receive() int {
	numRead, err := j.c.Read(j.buf)
	if err == io.EOF {
		return 0
	}
	if err != nil {
		log.Fatal(err)
	}
	return numRead
}

func (j *Jeb) parse(numRead int) (filename string, line int, function string) {
	var err error
	parts := strings.Split(string(j.buf[:numRead]), ":")
	if len(parts) != 3 {
		log.Fatalf("Expected 3 parts, got %v", parts)
	}
	filename = parts[0]
	line, err = strconv.Atoi(parts[1])
	if err != nil {
		log.Fatal(err)
	}
	function = parts[2]

	return filename, line, function
}

// proceed tells client to continue to next line
func (j *Jeb) proceed() {
	j.c.Write([]byte{'\n'})
}

func (j *Jeb) waitForInput() (rune, error) {
	ev := termbox.PollEvent()
	for ev.Type != termbox.EventKey {
		ev = termbox.PollEvent()
		log.Println("Event is", ev)
	}
	if ev.Key == termbox.KeyCtrlC {
		return 0, errors.New("Ctrl-C")
	}
	return ev.Ch, nil
}

func (j *Jeb) display(filename string, lineNum int) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	lines, ok := j.files[filename]
	if !ok {
		j.load(filename)
		lines, _ = j.files[filename]
	}

	for i, line := range lines {
		if i+1 == lineNum {
			tbprint(0, i, termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault, line)
		} else {
			out(0, i, line)
		}
	}
	termbox.Flush()
}

// load filename as array of file lines into Jeb.files
func (j *Jeb) load(filename string) {
	filedata, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(filedata), "\n")

	j.files[filename] = lines
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
