package main

import (
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Frontend string   `yaml:"frontend"`
	Backends []string `yaml:"backends"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	b, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Println(err)
		return
	}

	var conf Config
	err = yaml.Unmarshal(b, &conf)
	if err != nil {
		log.Println(err)
		return
	}

	l, err := net.Listen("tcp", conf.Frontend)
	if err != nil {
		log.Println(err)
		return
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
		}
		go handleConnection(conn, conf)
	}
}

func handleConnection(conn net.Conn, conf Config) {
	defer conn.Close()
	var err error
	var bck string
	var bckConn net.Conn
	maxLoop := len(conf.Backends) //Max loop limit
	loopCount := 0

	for { //Try until successful connection
		bck = conf.Backends[rand.Intn(len(conf.Backends))]
		bckConn, err = net.Dial("tcp", bck)
		if err != nil {
			log.Println(err)
			if maxLoop >= loopCount {
				log.Println("Loop limit reached!")
				return
			}
			loopCount++
		} else {
			break //Conn ok, break the loop
		}
	}

	go io.Copy(bckConn, conn)
	io.Copy(conn, bckConn)
}
