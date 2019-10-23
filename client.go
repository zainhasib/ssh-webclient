package main

import (
	"fmt"
	"log"
	"os"
	"io"
	"io/ioutil"
	"golang.org/x/crypto/ssh"
	"github.com/google/uuid"
)

type Node struct {
	User string
	Password string
}

func NewNode(user, password string) *Node {
	node := new(Node)
	node.User = user
	node.Password = password
	return node
}

func (node *Node) Conn(addr string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
        User: node.User,
        Auth: []ssh.AuthMethod{
            ssh.Password(node.Password),
        },
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    }

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", addr), config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

type Client struct {
	Color bool
	Stat bool
	Addr string
	User string
	Password string
	In io.WriteCloser
	Out io.Reader
	Session *ssh.Session
	Conn *ssh.Client
	File *os.File
	FilePath string
	Timeout int64
}

func NewClient() *Client {
	client := new(Client)
	client.Stat = false
	client.Addr = "localhost"
	client.User = "dev"
	client.Password = "password"
	client.Color = true
	return client
}

func (client *Client) IsConnected() bool {
	return client.Stat
}

func (client *Client) Disconnect() {
	client.Stat = false
	if err := client.In.Close(); err != nil {
		fmt.Println("in->" + err.Error())
	}
	if err := client.Session.Close(); err != nil {
		fmt.Println("session->" + err.Error())
	}
	if err := client.File.Close(); err != nil {
		fmt.Println("file->" + err.Error())
	}
	if err := client.Conn.Close(); err != nil {
		fmt.Println("conn->" + err.Error())
	}

	go func(path string) {
		for {
			err := os.Remove(client.FilePath)
			if err == nil {
				break
			}
		}
	}(client.FilePath)
}

func (client *Client) SendCmd(cmd string) {
	client.In.Write([]byte(cmd))
}

func (client *Client) GetOutFile() []byte {
	fia, _ := os.Open(client.FilePath)
	fda, _ := ioutil.ReadAll(fia)
	fia.Truncate(int64(len(fda)))
	return fda
}

func (client *Client) GetOutput() ([]byte, int) {
	buf := make([]byte, 4096)
    n, err := client.Out.Read(buf)
    if err != nil {
        log.Print(err)
    	return nil, 0
    }
	return buf, n
}

func (client *Client) Connect(addr, user, password string, handler func(client *Client, err error)) {
	client.User = user
	client.Password = password
	client.Addr = addr

	node := NewNode(client.User, client.Password)
	conn, err := node.Conn(client.Addr)
	if err != nil {
		log.Printf("Connect SSH exception(%s)", err.Error())
		handler(client, err)
		client.Disconnect()
		return
	}
	defer conn.Close()
	client.Conn = conn

	log.Printf("Connect SSH(%s) success", client.Addr)

	session, err := conn.NewSession()
	if err != nil {
		log.Printf("Connect SSH exception(%s)", err.Error())
		handler(client, err)
		return
	}
	defer session.Close()

	client.FilePath = "/tmp/" + uuid.New().String()
	file, _ := os.Create(client.FilePath)
	//session.Stdout = file
	client.File = file
	client.In, err = session.StdinPipe()
	if err != nil {
		log.Printf("Connect SSH exception(%s)", err.Error())
		handler(client, err)
		client.Disconnect()
		return
	}
	client.Out, err = session.StdoutPipe()
	if err != nil {
		log.Printf("Connect SSH exception(%s)", err.Error())
		handler(client, err)
		client.Disconnect()
		return
	}

	if client.Color {
		modes := ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}

		if err := session.RequestPty("xterm-256color", 29, 140, modes); err != nil {
			handler(client, err)
			panic(err)
		}
	}

	if err := session.Shell(); err != nil {
		log.Printf("Connect SSH exception(%s)", err.Error())
		handler(client, err)
		client.Disconnect()
		return
	}

	client.Stat = true
	client.Session = session
	handler(client, nil)

	if err := client.Session.Wait(); err == nil {
		fmt.Println("session disconnect")
		client.Disconnect()
	}
}
