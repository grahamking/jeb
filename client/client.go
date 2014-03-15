package client

import (
	"log"
	"net"
	"strings"
)

var c net.Conn

func init() {
	var err error
	c, err = net.Dial("unix", "/tmp/jeb-socket")
	if err != nil {
		log.Fatal(err)
	}
}

func Trace(args ...string) {
	c.Write([]byte(strings.Join(args, ":") + "\n"))
	in := make([]byte, 1)
	c.Read(in)
}
