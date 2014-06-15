// Build: go build jeb/example/b64
// Example usage:
//  dd if=/dev/urandom count=1 bs=64 status=none | ./b64
package main

import (
	"fmt"
	"io/ioutil"
	"jeb/example/b64lib"
	"log"
	"os"
)

func main() {

	indata, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	encoded := b64lib.Encode(indata)
	fmt.Println(encoded)
}
