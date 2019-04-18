package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const INDICATOR_IDLE = byte(0xF1)
const INDICATOR_DATA = byte(0xF2)
const RESERVE_BYTE = byte(0x00)
const END_BYTE = byte(0xFF)

var IDLE_MESSAGE = []byte{INDICATOR_IDLE, RESERVE_BYTE, END_BYTE}

var conn net.Conn

type Task struct {
	SystemReceiver string
	SystemSender   string
	TaskCode       string
	TypeCode       string
	Params         []string
	Data           string
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {

	case "GET":
		for k, v := range r.URL.Query() {
			fmt.Printf("%s: %s\n", k, v)
		}

		query := r.URL.Query()
		command := query.Get("command")

		// 004000F101F10280

		sendMessage(command, conn)
		w.Write([]byte("Received a GET request..\nSending Command: " + command))

	case "POST":
		reqBody, err := ioutil.ReadAll(r.Body)

		if err != nil {
			fmt.Printf("%v\n", err)
		}
		//print all of the POST data
		fmt.Printf("%s\n", reqBody)

		//convert byte array into a string for manipulation
		s := string(reqBody[:])

		//strip off bits at the beginning and end of the newly formatted string so you only have the context you want
		sFmt := s[186 : len(s)-245]

		//Replace all + values with a space " "
		fmt.Println(strings.Replace(sFmt, "+", " ", -1))

		//convert both static text and text message to ASCII hex
		back := []byte(sFmt)
		back2 := (hex.EncodeToString([]byte(back)))

		front := "00600090F151069200"

		//concationate the two
		ICX := front + back2
		//ICX variable will be teh one sent to the Commend Server driver
		fmt.Println(ICX)
		sendMessage(ICX, conn)
		//send a message back to SMS device or POST reciever to say message has been recieved.
		//w.Write([]byte("Message Sent to Commend Intercom\n"))

	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
	}

}

func sendMessage(text string, conn net.Conn) {
	//Cut off any newline character
	text = strings.Trim(text, "\r\n")

	//Turn string into bytes
	textBytes := []byte(text)

	//Start the array with indicator and reserve
	byteArray := []byte{INDICATOR_DATA, RESERVE_BYTE}

	//Then add the actual data
	byteArray = append(byteArray, textBytes...)

	//Then add the end byte
	byteArray = append(byteArray, END_BYTE)

	//Just a message for debugging
	log(fmt.Sprintf("About to send: % x", byteArray))

	//send bytes to server
	conn.Write(byteArray)
}

func waitForMessage(conn net.Conn) {
	message, _ := bufio.NewReader(conn).ReadBytes(END_BYTE)

	if bytes.Equal(message, IDLE_MESSAGE) {
		log("Received from server: IDLE")
		conn.Write(IDLE_MESSAGE)
		log("Sent to server: IDLE")

	}

	waitForMessage(conn)
}

func log(msg string) {
	fmt.Printf("[%s] %v\n", time.Now().Format("02/01/2006 15:04:05.000"), msg)
}

func main() {
	log("Connecting...")

	var err error
	conn, err = net.Dial("tcp", "10.0.20.4:17000")
	if err != nil {
		panic(err)
	}

	log("Connected.")

	http.HandleFunc("/", helloWorld)
	go http.ListenAndServe(":9999", nil)

	log("Web server started.")

	go waitForMessage(conn)

	log("Waiting for IDLEs.")

	for {
		// read in input from command line
		reader := bufio.NewReader(os.Stdin)
		// Read everything until you press enter
		text, _ := reader.ReadString('\n')
		sendMessage(text, conn)
	}
}
