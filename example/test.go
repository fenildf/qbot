package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"

	"github.com/forthxu/qbot"
)

func execCommand(path string) {
	c := exec.Command("cmd", "/C", "start", path)
	if err := c.Run(); err != nil {
		//fmt.Println("Error: ", err)
	}
}

func main() {
	r, _ := qbot.New()
	r.OnQRChange(func(r *qbot.Robot, qrdata []byte) {
		if err := ioutil.WriteFile("v.png", qrdata, 0666); err == nil {
			execCommand("v.png")
		}
	})
	r.OnMessage(func(r *qbot.Robot, message *qbot.Message) {
		fmt.Println(message)

		switch message.PollType {
		case "message":
			r.SendToBuddy(message.FromUin, message.Content+"\r\n\t--qbot")
		case "group_message":
			r.SendToGroup(message.FromUin, message.Content)
		case "discu_message":
			r.SendToDiscuss(message.FromUin, message.Content)
		}
	})
	r.Run()
}