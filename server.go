package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/nsf/termbox-go"
)

const (
	STACK_LINE = 0
	BODY_START = 2
)

func runServer() {

	l, err := net.Listen("unix", "/tmp/jeb-socket")
	if err != nil {
		log.Println(err)
		return
	}
	defer l.Close()

	fmt.Println("Waiting for message from client...")

	c, err := l.Accept()
	if err != nil {
		log.Println(err)
		return
	}
	defer c.Close()

	j := NewJeb(c, keyLoop())
	j.run() // doesn't return until exit
}

func setupLogging() {
	log.Println("Logging to jeb.log")
	l, err := os.Create("jeb.log")
	if err != nil {
		log.Println(err)
		return
	}
	log.SetOutput(l)
}

// Runs a go-routine to fetch keyboard events
func keyLoop() <-chan termbox.Event {
	ch := make(chan termbox.Event)
	go func() {
		var ev termbox.Event
		for {
			ev = termbox.PollEvent()
			if ev.Type == termbox.EventKey {
				ch <- ev
			}
		}
	}()
	return ch
}

type Jeb struct {
	c         net.Conn
	buf       []byte
	files     map[string][]string
	stack     *stack
	contLevel int
	keyChan   <-chan termbox.Event
}

func NewJeb(c net.Conn, keyChan <-chan termbox.Event) *Jeb {
	return &Jeb{
		c:         c,
		buf:       make([]byte, 1024),
		files:     make(map[string][]string),
		stack:     newStack(),
		contLevel: -1,
		keyChan:   keyChan,
	}
}

func (j *Jeb) run() {
	termbox.Init()
	defer termbox.Close()
	setupLogging()

	for {
		numRead := j.receive()
		if numRead == 0 {
			break
		}

		filename, line, function := j.parse(numRead)
		if filename == "" {
			j.proceed()
			continue
		}

		j.stack.condPush(function)

		if j.contLevel == -1 || j.contLevel < j.stack.Pos {
			j.display(filename, line)

			_, err := j.waitForInput()
			if err != nil {
				break
			}
			/*
				if input == 's' {
					// step in
					j.contLevel = -1
				} else if input == 'n' {
					// next (step over)
					j.contLevel = j.stack.Pos
				}
			*/

		}
		j.proceed()
	}
}

func (j *Jeb) receive() int {
	numRead, err := j.c.Read(j.buf)
	if err == io.EOF {
		return 0
	}
	if err != nil {
		log.Println(err)
		return -1
	}
	return numRead
}

// TODO: This should be able to return error as well
func (j *Jeb) parse(numRead int) (filename string, line int, function string) {
	var err error

	parts := strings.Split(string(j.buf[:numRead]), ":")
	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "ENTER":
		j.stack.push(args[0])
		return
	case "EXIT":
		j.stack.pop(args[0])
		return
	case "LINE":
	default:
		log.Println("Unknown command: %s\n", cmd)
	}

	if len(args) != 3 {
		log.Println("Expected 3 parts, got %v", args)
	}
	filename = args[0]
	line, err = strconv.Atoi(args[1])
	if err != nil {
		log.Println(err)
	}
	function = args[2]

	return filename, line, function
}

// proceed tells client to continue to next line
func (j *Jeb) proceed() {
	j.c.Write([]byte{'\n'})
}

func (j *Jeb) waitForInput() (rune, error) {
	ev := <-j.keyChan
	if ev.Key == termbox.KeyCtrlC {
		return 0, errors.New("Ctrl-C")
	}
	return ev.Ch, nil
}

func (j *Jeb) display(filename string, lineNum int) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	j.printStackLine()

	lines, ok := j.files[filename]
	if !ok {
		j.load(filename)
		lines, _ = j.files[filename]
	}

	_, h := termbox.Size()
	middle := h / 2
	lineStart := 0
	highlight := lineNum
	if lineNum > middle {
		lineStart = lineNum - middle
		highlight = middle
	}
	lineEnd := lineStart + h
	if lineEnd > len(lines) {
		lineEnd = len(lines)
	}

	for i, line := range lines[lineStart:lineEnd] {
		offset := i + BODY_START
		if i+1 == highlight {
			tbprint(0, offset, termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault, line)
		} else {
			out(0, offset, line)
		}
	}
	termbox.Flush()
}

// load filename as array of file lines into Jeb.files
func (j *Jeb) load(filename string) {
	filedata, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return
	}
	lines := strings.Split(string(filedata), "\n")

	j.files[filename] = lines
}

func (j *Jeb) printStackLine() {
	st := j.stack.stack()
	stackLine := make([]string, 0, len(st))
	for _, function := range st {
		stackLine = append(stackLine, function)
	}
	w, _ := termbox.Size()
	out(0, STACK_LINE, strings.Join(stackLine, "> "))
	out(0, STACK_LINE+1, strings.Repeat("-", w))
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

type stack struct {
	s   []string
	Pos int
}

func newStack() *stack {
	return &stack{
		s:   make([]string, 64),
		Pos: 0,
	}
}

func (s *stack) push(str string) {
	s.s[s.Pos] = str
	s.Pos++
}

func (s *stack) pop(str string) {
	s.Pos--
}

func (s *stack) stack() []string {
	return s.s[:s.Pos]
}

// condPush pushes 'str' on to the stack if it's not already the last element
func (s *stack) condPush(str string) {
	if s.Pos == 0 || s.s[s.Pos-1] != str {
		s.push(str)
	}
}
