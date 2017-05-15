package main

import (
    "encoding/json"
    "fmt"
    "net/url"
    "strconv"
    "strings"
    "time"

    "github.com/gorilla/websocket"
)

func genPingReply(time float64) string {
    pingReply := map[string]interface{}{
        "type": "ping-reply",
        "data": map[string]interface{}{
            "time": time,
        },
    }

    str, err := json.Marshal(pingReply)
    if err != nil { panic(err) }
    return string(str)
}

func genNickEvent(nick string) string {
    nickEvent := map[string]interface{}{
        "type": "nick",
        "data": map[string]interface{}{
            "name": nick,
        },
    }

    str, err := json.Marshal(nickEvent)
    if err != nil { panic(err) }
    return string(str)
}

func reply(message string, thisMessage map[string]interface{}, conn *websocket.Conn) {
    id := thisMessage["data"].(map[string]interface{})["id"].(string)
    response := fmt.Sprintf("{\"type\": \"send\", \"data\": {\"content\": \"%s\", \"parent\": \"%s\"}}", message, id)
    err := conn.WriteMessage(websocket.TextMessage, []byte(response))
    if err != nil { panic(err) }
}

func main() {
    u := url.URL{Scheme: "wss", Host: "euphoria.io", Path: "/room/xkcd/ws"}

    conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil { panic(err) }

    defer conn.Close()

    timers := make(chan string)

    for {
        _, message, err := conn.ReadMessage()
        if err != nil { panic(err) }
        var thisMessage map[string]interface{}
        json.Unmarshal(message, &thisMessage)
        msgType := thisMessage["type"]
        switch msgType{
        case "ping-event":
            //pong!
            time := thisMessage["data"].(map[string]interface{})["time"].(float64)
            response := genPingReply(time)
            err := conn.WriteMessage(websocket.TextMessage, []byte(response))
            if err != nil { panic(err) }
        case "snapshot-event":
            //Hello!
            response := genNickEvent("TimerBot")
            err := conn.WriteMessage(websocket.TextMessage, []byte(response))
            if err != nil { panic(err) }
        case "send-event":
            content := thisMessage["data"].(map[string]interface{})["content"]

            //Handles botrulez
            switch content{
            case "!help":
                reply("I offer timers! Use !timer <minutes>. Courtesy of Pouncy.", thisMessage, conn)
            case "!help @TimerBot":
                reply("I was written by Pouncy to offer timers. Use !timer <minutes> to be pinged after <minutes>.", thisMessage, conn)
            case "!ping":
                reply("Pong!", thisMessage, conn)
            case "!ping @TimerBot":
                reply("Pong!", thisMessage, conn)
            case "!kill @TimerBot":
                panic("Killed!")
            }

            //Handles timer requests
            if len(strings.Fields(content.(string))) > 1 && strings.Fields(content.(string))[0] == "!timer" {
                duration, err := strconv.Atoi(strings.Fields(content.(string))[1])
                if err != nil || duration < 1 || duration > 1440 {
                    reply("Sorry, I couldn't parse that request!", thisMessage, conn)
                } else {
                    sender := thisMessage["data"].(map[string]interface{})["sender"].(map[string]interface{})["name"].(string)
                    go func(duration int, sender string, timers chan string) {
                        timer := time.NewTimer(time.Minute * time.Duration(duration))
                        <-timer.C
                        timers <- sender
                    }(duration, sender, timers)
                    id := thisMessage["data"].(map[string]interface{})["id"].(string)
                    response := fmt.Sprintf("{\"type\": \"send\", \"data\": {\"content\": \"I'll ping you in %d minutes.\", \"parent\": \"%s\"}}", duration, id)
                    err = conn.WriteMessage(websocket.TextMessage, []byte(response))
                }
            }
        }

        select{
        case sender := <-timers:
            message := fmt.Sprintf("@%s, your timer is done!", strings.Join(strings.Fields(sender),""))
            response := "{\"type\": \"send\", \"data\": {\"content\": \"" + message + "\"}}"
            err = conn.WriteMessage(websocket.TextMessage, []byte(response))
        default:
        }
    }
}
