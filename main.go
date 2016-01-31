package main

import (
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Frontend string   `yaml:"frontend"`
	Backends []string `yaml:"backends"`
}

type Backend struct {
	Addr     string
	Network  string
	OpenConn int
	Alive    bool
	LastConn time.Time

	BackendLock sync.Mutex
}

func NewBackend(addr string) *Backend {
	return &Backend{
		Addr:        addr,
		Network:     "tcp",
		OpenConn:    0,
		Alive:       false,
		BackendLock: sync.Mutex{},
	}
}

func (b *Backend) UpdateAlive() {
	c, err := net.Dial(b.Network, b.Addr)
	if err != nil {
		b.BackendLock.Lock()
		b.Alive = false
		b.BackendLock.Unlock()
		return
	}
	c.Close()
	b.BackendLock.Lock()
	b.Alive = true
	b.BackendLock.Unlock()
}

func (b *Backend) UpdateConn(i int) {
	b.BackendLock.Lock()
	b.OpenConn += i
	b.BackendLock.Unlock()
}

func (b *Backend) UpdateLastConn() {
	b.BackendLock.Lock()
	b.LastConn = time.Now()
	b.BackendLock.Unlock()
}

var backends []*Backend

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
	for _, b := range conf.Backends {
		bck := NewBackend(b)
		bck.UpdateAlive()
		backends = append(backends, bck)
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
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	var err error
	var bck *Backend
	var bckConn net.Conn
	maxLoop := len(backends) //Max loop limit
	loopCount := 0

	for { //Try until successful connection

		bck = ChooseBackend(backends, "random")

		bckConn, err = net.Dial(bck.Network, bck.Addr)
		if err != nil {
			log.Println(err)
			if maxLoop >= loopCount {
				log.Println("Loop limit reached!")
				bck.UpdateAlive() //Maybe it's not alive
				return
			}
			loopCount++
		} else {
			break //Conn ok, break the loop
		}
	}
	bck.UpdateConn(1)
	bck.UpdateLastConn()
	defer LogStats(bck) //Defer in different order
	defer bck.UpdateConn(-1)
	defer bckConn.Close()
	LogStats(bck)

	go io.Copy(bckConn, conn)
	io.Copy(conn, bckConn)
}

func ChooseBackend(backends []*Backend, alg string) *Backend {
	switch alg {
	case "random":
		return backends[rand.Intn(len(backends))]
	}
	return nil
}

func LogStats(bck *Backend) {
	//for _, bck := range backends {
	log.Printf("%s: Connected: %d, IsAlive: %t, LastConnection: %s", bck.Addr, bck.OpenConn, bck.Alive, bck.LastConn.String())
	//}
}
