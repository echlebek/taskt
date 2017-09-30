// taskt executes the task it is taskt with
package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
)

var (
	bind = flag.String("port", ":3000", "Address to bind to")
)

func main() {
	ln, err := net.Listen("tcp", *bind)
	if err != nil {
		log.Fatal(err)
	}
	server := NewTaskServer(ln)
	defer server.Close()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
