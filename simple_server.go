package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
)

const (
	CONN_HOST = "localhost"
	CONN_PORT = "3333"
	CONN_TYPE = "tcp"
)

type Account struct {
	Password string
	User     string
	Nickname string
}

type Channel struct {
	Name      string
	Topic     string
	Key       string
	AdminList []*Account
	UserList  []*Account
	BanList   []*Account
	UserMap   map[string]*Account
}

type Env struct {
	ChannelList []*Channel
	AccountList []*Account
	UserMap     map[string]*Account
	NicknameMap map[string]*Account
	ConnMap     map[string]net.Conn
	ChannelMap  map[string]*Channel
}

type Session struct {
	Env     *Env
	Conn    net.Conn
	Account *Account
}

func main() {
	ln, err := net.Listen(CONN_TYPE, CONN_HOST+":"+CONN_PORT)
	if err != nil {
		fmt.Println("Error listening :", err.Error())
		os.Exit(1)
	}
	defer ln.Close()
	env := Env{AccountList: []*Account{},
		ChannelList: []*Channel{},
		UserMap:     make(map[string]*Account),
		NicknameMap: make(map[string]*Account),
		ConnMap:     make(map[string]net.Conn),
		ChannelMap:  make(map[string]*Channel)}
	fmt.Println("Listening on ", CONN_HOST+":"+CONN_PORT)
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting ", err.Error())
			continue
		}
		log.Println("accept connection", conn)
		go runSession(&env, conn)
	}
}

func runSession(env *Env, conn net.Conn) {
	session := Session{Env: env, Conn: conn, Account: nil}
	defer session.closeConnection()

	if !session.authorize() {
		return
	}
	defer session.disconnect()

	log.Println("ConnMap", session.Env.ConnMap)

	for {
		request, err := session.getRequest()
		if err != nil {
			break
		}
		session.handleRequest(request)
	}
}

func (session *Session) closeConnection() {
	log.Println("close session", session.Conn)
	session.Conn.Close()
}

func (session *Session) disconnect() {
	log.Println("disconnect", session.Account.Nickname)
	delete(session.Env.ConnMap, session.Account.Nickname)
}

func (session *Session) handleRequest(request string) {
	switch {
	case regexp.MustCompile(`^(:(.+) +)?NICK`).MatchString(request):
		session.changeNickname(request)
	case strings.HasPrefix(request, "PRIVMSG "):
		session.privateMSG(request)
	case strings.Contains(request, "JOIN"):
		session.joinChan(request)
	case strings.Contains(request, "PART"):
		session.leaveChan(request)
	case request == "NAMES":
		session.cmdNAMES()
	case request == "LIST":
		session.cmdLIST()
	}
}

func (session *Session) getRequest() (string, error) {
	request := make([]byte, 1024)
	len, err := session.Conn.Read(request)
	if err != nil {
		log.Println("Error reading: ", err)
		return "", err
	}
	if request[len-2] != '\r' || request[len-1] != '\n' {
		return "", errors.New("no CRLF")
	}
	requestStr := string(request[:len-2])
	fmt.Println("<" + requestStr + ">")
	return requestStr, nil
}

func (session *Session) cmdNAMES() {
	list := make([]string, len(session.Env.ConnMap))
	for key := range session.Env.ConnMap {
		list = append(list, key)
	}
	message := fmt.Sprintf("%s\r\n", list)
	session.Conn.Write([]byte(message))
}

func (session *Session) cmdLIST() {
	list := make([]string, len(session.Env.ChannelMap))
	for key := range session.Env.ChannelMap {
		list = append(list, key)
	}
	message := fmt.Sprintf("%s\r\n", list)
	session.Conn.Write([]byte(message))
}
