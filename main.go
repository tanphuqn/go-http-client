package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type KernelHeaderResponse struct {
	MsgId    string `json:"msg_id"`
	MsgType  string `json:"msg_type"`
	Username string `json:"username"`
	Session  string `json:"session"`
	// Date     time.Time `json:"date"`
	Version string `json:"version"`
}
type KernelContentResponse struct {
	Name           string `json:"name"`
	Text           string `json:"text"`
	Code           string `json:"code"`
	ExecutionCount int32  `json:"execution_count"`
}

type KernelMessageResponse struct {
	Header       KernelHeaderResponse     `json:"header"`
	MsgId        string                   `json:"msg_id"`
	MsgType      string                   `json:"msg_type"`
	ParentHeader KernelHeaderResponse     `json:"parent_header"`
	Metadata     map[string]interface{}   `json:"metadata"`
	Content      KernelContentResponse    `json:"content"`
	Buffers      []map[string]interface{} `json:"buffers"`
	Channel      string                   `json:"channel"`
}

type KernelSpec struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	KernelPath  string `json:"kernel_path"`
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	// https://github.com/gorilla/websocket/blob/master/examples/echo/client.go

	kernelId := "519d1cf2-2db4-4486-9872-52e66e5a7eae"
	uuidMsg, _ := uuid.NewUUID()
	msgId := uuidMsg.String()
	log.Println("msgId:", msgId)
	uuidSession, _ := uuid.NewUUID()
	session := uuidSession.String()
	code := "!jupyter kernelspec list"
	log.Println("msgId:", msgId)
	log.Println("session:", session)

	var kernels []KernelSpec

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	// url := "wss://jupyterhub.cluster-dev.exspanse.com/user/clearuser-1/api/kernels/c9d53002-b511-4052-a44b-c382ab4d0189/channels?token=4d2dd9c82e624fd4afd074b265e44f73"
	url := "wss://jupyterhub.cluster-dev.exspanse.com/user/user-1/api/kernels/" + kernelId + "/channels?token=4d2dd9c82e624fd4afd074b265e44f73"

	log.Printf("connecting to %s", url)
	//wss://jupyterhub.cluster-dev.exspanse.com/user/user-1/api/kernels/996e2a47-a9e8-4183-9895-cff65ecbcaab/channels?token=4d2dd9c82e624fd4afd074b265e44f73 –
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			var kernelSpecs KernelMessageResponse
			if err := json.Unmarshal(message, &kernelSpecs); err != nil {
				log.Println("read:", err)
				return
			}
			if kernelSpecs.MsgType == "stream" &&
				session == kernelSpecs.ParentHeader.Session &&
				msgId == kernelSpecs.ParentHeader.MsgId {
				// log.Println("recv: ", string(message))
				// log.Println("kernelSpecs: ", kernelSpecs.Content.Text)
				kernelLines := strings.Split(kernelSpecs.Content.Text, "\r\n")
				for _, line_str := range kernelLines {
					if strings.HasPrefix(line_str, "Available kernels") {
						continue
					}
					cols := strings.Split(line_str, " ")
					var kernelCols []string

					for _, col := range cols {
						if col != "" {
							kernelCols = append(kernelCols, col)
						}
					}

					if len(kernelCols) > 1 {
						name := kernelCols[0]
						pkg := kernelCols[1]
						// log.Println(name, "name")
						// jsonFile, err := os.Open(pkg + "/kernel.json")
						// if err != nil {
						// 	continue
						// }
						// byteValue, _ := ioutil.ReadAll(jsonFile)
						// var result map[string]interface{}
						// json.Unmarshal([]byte(byteValue), &result)
						kernel := KernelSpec{
							Name: name,
							// DisplayName: fmt.Sprintf("%s", result["display_name"]),
							KernelPath: pkg,
						}
						kernels = append(kernels, kernel)
					}

				}
				log.Println("kernels: ", kernels)
				break
			}

		}
	}()

	// ticker := time.NewTicker(time.Second)
	// defer ticker.Stop()

	jsonData := map[string]interface{}{
		"header": map[string]string{
			"msg_id":   msgId,
			"username": "user-1",
			// "username": "exspanse",
			"session":  session,
			"msg_type": "execute_request",
			"version":  "5.2",
		},
		"msg_type": "execute_request",
		"metadata": map[string]string{},
		"content": map[string]interface{}{
			"code":             code,
			"silent":           false,
			"store_history":    true,
			"user_expressions": map[string]string{},
			"allow_stdin":      true,
			"stop_on_error":    true,
		},
		"buffers":       nil,
		"parent_header": map[string]string{},
		"channel":       "shell",
	}

	jsonB, err := json.Marshal(jsonData)
	c.WriteMessage(websocket.TextMessage, jsonB)
	if err != nil {
		log.Println("write:", err)
		return
	}

	for {
		select {
		case <-done:
			return
			// case t := <-ticker.C:

			// case <-interrupt:
			// 	log.Println("interrupt")

			// 	// Cleanly close the connection by sending a close message and then
			// 	// waiting (with timeout) for the server to close the connection.
			// 	err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			// 	if err != nil {
			// 		log.Println("write close:", err)
			// 		return
			// 	}
			// 	select {
			// 	case <-done:
			// 	case <-time.After(time.Second):
			// 	}
			// 	return
		}
	}

}
