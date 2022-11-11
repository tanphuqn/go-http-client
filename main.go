package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "github.com/snowflakedb/gosnowflake"
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

type KernelMessage struct {
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

type Env struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	KernelPath  string `json:"kernel_path"`
}

func RunConsoleMessages(kernelId string, username string, token string, cmd string) (string, error) {
	uuidMsg, _ := uuid.NewUUID()
	msgId := uuidMsg.String()
	uuidSession, _ := uuid.NewUUID()
	session := uuidSession.String()
	log.Println("msgId:", msgId)
	log.Println("session:", session)
	result := ""
	url := "wss://jupyterhub.cluster-dev.exspanse.com/user/user-1/api/kernels/" + kernelId + "/channels?token=" + token

	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	log.Printf("connecting to %s", url)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			var kernelSpecs KernelMessage
			if err := json.Unmarshal(message, &kernelSpecs); err != nil {
				log.Println("read:", err)
				return
			}
			if kernelSpecs.MsgType == "stream" &&
				session == kernelSpecs.ParentHeader.Session &&
				msgId == kernelSpecs.ParentHeader.MsgId {
				result = kernelSpecs.Content.Text
				// log.Println(result, "content")
				break
			}
		}
		return
	}()

	jsonData := map[string]interface{}{
		"header": map[string]string{
			"msg_id":   msgId,
			"username": username,
			// "username": "exspanse",
			"session":  session,
			"msg_type": "execute_request",
			"version":  "5.2",
		},
		"msg_type": "execute_request",
		"metadata": map[string]string{},
		"content": map[string]interface{}{
			"code":             fmt.Sprintf("!%s", cmd),
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
		return "", err
	}
	// log.Printf("WriteMessage to %s", string(jsonB))
	for {
		select {
		case <-done:
			// log.Println(result, "result")
			return result, nil
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

func ConvertToKernelSpecObject(kernelId string, username string, token string, content string) (kernels []KernelSpec, err error) {
	kernelLines := strings.Split(content, "\r\n")
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
			code := fmt.Sprintf("cat %s/kernel.json", pkg)
			kernelJson, err := RunConsoleMessages(kernelId, username, token, code)
			if err != nil {
				log.Println("Error :", err)
				continue
			}

			var result map[string]interface{}
			json.Unmarshal([]byte(kernelJson), &result)
			kernel := KernelSpec{
				Name:        name,
				DisplayName: fmt.Sprintf("%s", result["display_name"]),
				KernelPath:  pkg,
			}
			kernels = append(kernels, kernel)
		}
	}
	return kernels, nil
}

func ConvertToEnvObject(kernelId string, username string, token string, content string) (envs []Env, err error) {
	kernelLines := strings.Split(content, "\r\n")
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
			code := fmt.Sprintf("cat %s/kernel.json", pkg)
			kernelJson, err := RunConsoleMessages(kernelId, username, token, code)
			if err != nil {
				log.Println("Error :", err)
				continue
			}

			var result map[string]interface{}
			json.Unmarshal([]byte(kernelJson), &result)
			env := Env{
				Name:        name,
				DisplayName: fmt.Sprintf("%s", result["display_name"]),
				KernelPath:  pkg,
			}
			envs = append(envs, env)
		}
	}
	return envs, nil
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	// https://github.com/gorilla/websocket/blob/master/examples/echo/client.go

	// kernelId := "f0bf08b5-583c-4879-b228-5278e4614f71"
	// username := "user-1"
	// token := "4d2dd9c82e624fd4afd074b265e44f73"
	// code := "jupyter kernelspec list"
	// content, err := RunConsoleMessages(kernelId, username, token, code)
	// if err != nil {
	// 	log.Println("Error :", err)
	// 	return
	// }

	// kernels, err := ConvertToKernelSpecObject(kernelId, username, token, content)
	// log.Println(kernels, "kernels")

	// First task is "Snowflake API integration"
	// https://docs.snowflake.com/en/user-guide/go-driver.html
	// https://pkg.go.dev/github.com/snowflakedb/gosnowflake#section-readme
	// go get -u github.com/snowflakedb/gosnowflake
	// https://docs.snowflake.com/en/sql-reference/intro-summary-data-types.html

	db, err := sql.Open("snowflake", "tanphuqn:@Minh10112014@pxwdfsd-vy07669/EXSPANSE/EX_SCHEMA?warehouse=wh_exspanse")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("insert into users values (?)", "aldy")

	fmt.Println("Connect to success")
	defer db.Close()

}
