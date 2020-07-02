package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"testproxy/mysql"
)

var lock sync.Mutex
var trueList []string
var ip string
var list string

func main() {
	flag.StringVar(&ip, "l", ":3306", "-l=0.0.0.0:9897 指定服务监听的端口")
	flag.StringVar(&list, "d", "172.16.0.6:3306", "-d=127.0.0.1:1789,127.0.0.1:1788 指定后端的IP和端口,多个用','隔开")
	flag.Parse()
	trueList = strings.Split(list, ",")
	if len(trueList) <= 0 {
		fmt.Println("后端IP和端口不能空,或者无效")
		os.Exit(1)
	}
	server()
}

func server() {
	lis, err := net.Listen("tcp", ip)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer lis.Close()
	for {
		conn, err := lis.Accept()
		if err != nil {
			fmt.Println("建立连接错误:%v\n", err)
			continue
		}
		fmt.Println(conn.RemoteAddr(), conn.LocalAddr())
		go handle(conn)
	}
}

func handle(sconn net.Conn) {
	defer sconn.Close()
	ip, ok := getIP()
	if !ok {
		return
	}
	dconn, err := net.Dial("tcp", ip)
	if err != nil {
		fmt.Printf("连接%v失败:%v\n", ip, err)
		return
	}

	spkt := mysql.NewPacketIO(sconn)
	dpkt := mysql.NewPacketIO(dconn)



	go func(sconn net.Conn, dconn net.Conn) {
		for {
			fmt.Println("wait recv from sconn")
			data, err := spkt.ReadPacket()
			if err != nil {
				panic(err)
			}
			fmt.Println(len(data))
			fmt.Println("wait send to dconn")
			err = dpkt.WritePacket(data)
			if err != nil {
				panic(err)
			}
		}
	}(sconn, dconn)
	go func(sconn net.Conn, dconn net.Conn) {
		for {
			fmt.Println("wait recv from dconn")
			data, err := dpkt.ReadPacket()
			if err != nil {
				panic(err)
			}
			fmt.Println(len(data))
			fmt.Println("wait send to sconn")
			err = spkt.WritePacket(data)
			if err != nil {
				panic(err)
			}
		}
	}(sconn, dconn)

	ch := make(chan bool, 1)
	<-ch

	dconn.Close()
}

func getIP() (string, bool) {
	lock.Lock()
	defer lock.Unlock()

	if len(trueList) < 1 {
		return "", false
	}
	ip := trueList[0]
	trueList = append(trueList[1:], ip)
	return ip, true
}
