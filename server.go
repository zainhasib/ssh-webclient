package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
    	return true
    },
}

func reader(ws *websocket.Conn, client *Client) {
	defer ws.Close()
	for {
		buf := make([]byte, 4096)
        n, err := client.Out.Read(buf)
        if err != nil {
            log.Print(err)
        	return
        }
		err = ws.WriteMessage(websocket.TextMessage, buf[:n])
        if err != nil {
        	log.Print(err)
        	return
        }
	}
}

func writer(ws *websocket.Conn, client *Client) {
	defer ws.Close()
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", string(message))
		client.SendCmd(string(message))
	}
}


func ReadUsernameTillNextLine(ws *websocket.Conn) (string) {
	str := ""
	for {
		mt, message, err := ws.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		ws.WriteMessage(mt, message)
		if []rune(string(message))[0] == 13 {
			ws.WriteMessage(mt, []byte("\r\n"))
			log.Printf("end reached")
			break
		}
		fmt.Println([]rune(string(message))[0])
		str += string(message)
		log.Println("recv: " + string(message[:]))
	}
	return str
}

func ReadPasswordTillNextLine(ws *websocket.Conn) (string) {
	str := ""
	for {
		mt, message, err := ws.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		if []rune(string(message))[0] == 13 {
			ws.WriteMessage(mt, []byte("************\r\n\n"))
			log.Printf("end reached")
			break
		}
		fmt.Println([]rune(string(message))[0])
		str += string(message)
		log.Println("recv: " + string(message[:]))
	}
	return str
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	ws, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	ws.WriteMessage(websocket.TextMessage, []byte("> Enter username : "))
	username := ReadUsernameTillNextLine(ws)
	ws.WriteMessage(websocket.TextMessage, []byte("> Enter password : "))
	password := ReadPasswordTillNextLine(ws)
	client := NewClient()
	client.User = username
	client.Password = password
	go client.Connect(client.Addr, client.User, client.Password, func(client *Client, err error) {
		if err != nil {
			fmt.Println(err.Error())
			ws.WriteMessage(websocket.TextMessage, []byte(err.Error()))
			ws.Close()
			return
		}
		go reader(ws, client)
		writer(ws, client)
	})
}

func main() {
	log.SetFlags(0)
	http.Handle("/", http.FileServer(http.Dir("./app/")))
	http.HandleFunc("/ws", serveWs)
	log.Fatal(http.ListenAndServeTLS(":8080", "server.crt", "server.key", nil))
}
