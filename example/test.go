package main

import (
	"fmt"
	//"io/ioutil"
	"os/exec"

	//"github.com/forthxu/qbot"
)

func execCommand(path string) {
	c := exec.Command("cmd", "/C", "start", path)
	if err := c.Run(); err != nil {
		//fmt.Println("Error: ", err)
	}
}
/*
function hash(b, i) {
	for (var a = [], s = 0; s < i.length; s++)
		a[s % 4] ^= i.charCodeAt(s);

	console.log(a)
	var j = ["EC", "OK"], d = [];
	d[0] = b >> 24 & 255 ^ j[0].charCodeAt(0);
	d[1] = b >> 16 & 255 ^ j[0].charCodeAt(1);
	d[2] = b >> 8 & 255 ^ j[1].charCodeAt(0);
	d[3] = b & 255 ^ j[1].charCodeAt(1);
	console.log(d)

	j = [];
	for (s = 0; s < 8; s++)
		j[s] = s % 2 == 0 ? a[s >> 1] : d[s >> 1];
	console.log(jj)

	a = ["0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F"];
	d = "";
	for (s = 0; s < j.length; s++)
		d += a[j[s] >> 4 & 15], d += a[j[s] & 15];
	return d;
}
hash(472767085,"")
*/
func hash(b int, i string) string {
	a := [4]int{}
	for s := 0; s < len(i); s++{
		a[s%4] ^= int(i[s])
	}
	
	j := []string{"EC", "OK"}
	d := []int{b >> 24 & 255 ^ int(j[0][0]), b >> 16 & 255 ^ int(j[0][1]),b >> 8 & 255 ^ int(j[1][0]),b & 255 ^ int(j[1][1])}

	jj := [8]int{}
	for s := 0; s < 8; s++{
		if s%2==0 {
			jj[s] = a[s >> 1]
		}else{
			jj[s] = d[s >> 1]
		}
	}	
	
	aa := [16]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F"}
	dd := ""
	for s := 0; s < len(jj); s++{
		dd += aa[jj[s] >> 4 & 15]
		dd += aa[jj[s] & 15]
	}
	
	return dd;
}

func main() {

	hash(472767085,"")

	fmt.Println('a')

	/*
	r.OnQRChange(func(r *qbot.Robot, qrdata []byte) {
		if err := ioutil.WriteFile("v.png", qrdata, 0666); err == nil {
			execCommand("v.png")
		}
	})
	r.OnMessage(func(r *qbot.Robot, message *qbot.Message) {
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
	*/
}