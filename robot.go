package qbot

import (
	"fmt"
	"github.com/bitly/go-simplejson"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	//"os"
	//"github.com/mdp/qrterminal"
)

type H map[string]string

type Robot struct {
	client       *http.Client
	uid 		 int
	onQRChange   func(*Robot, []byte)
	onCheckLogin func(*Robot) bool
	onLogin      func(*Robot)
	onMessage    func(*Robot, *Message)
	parameter    H
	header       H
}

type Message struct {
	PollType string `json:"poll_type"`
	Content  string `json:"content"`
	FromUin  int    `json:"from_uin"`
	SendUin  int    `json:"send_uin"`
	MsgId    int    `json:"msg_id"`
	MsgType  int    `json:"msg_type"`
	Time     int    `json:"time"`
	ToUin    int    `json:"to_uin"`
	Atable   bool
}

func New() (*Robot, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &Robot{
		client: &http.Client{
			Jar: jar,
		},
		header:    H{},
		parameter: H{},
	}, nil
}

func (r *Robot) OnQRChange(fun func(*Robot, []byte)) {
	r.onQRChange = fun
}

func (r *Robot) OnCheckLogin(fun func(*Robot) bool) {
	r.onCheckLogin = fun
}

func (r *Robot) OnLogin(fun func(*Robot)) {
	r.onLogin = fun
}

func (r *Robot) OnMessage(fun func(*Robot, *Message)) {
	r.onMessage = fun
}

func (r *Robot) Run() {
	if r.onCheckLogin == nil || !r.onCheckLogin(r) {
		qrurl := "https://ssl.ptlogin2.qq.com/ptqrshow?appid=501004106&e=2&l=M&s=3&d=72&v=4&t=0.17856508562994167&daid=164"

		qrdata, err := r.Get(qrurl)
		if err != nil {
			log.Fatal(err)
		}
		if r.onQRChange != nil {
			r.onQRChange(r, qrdata)
		}
		qrsig := r.GetCookie(qrurl, "qrsig")
		ticker := time.NewTicker(time.Second * 1)
		regexp_image_state := regexp.MustCompile(`ptuiCB\(\'(\d+)\'`)

		validate_login:
			for range ticker.C {
				ptqrlogin := "https://ssl.ptlogin2.qq.com/ptqrlogin?u1=http%3A%2F%2Fw.qq.com%2Fproxy.html&ptqrtoken=" + r.GetToken(qrsig) + "&ptredirect=0&h=1&t=1&g=1&from_ui=1&ptlang=2052&action=0-0-" + r.GetTimestamp() + "&js_ver=10228&js_type=1&login_sig=&pt_uistyle=40&aid=501004106&daid=164&mibao_css=m_webqq&"
				logindata, err := r.Get(ptqrlogin)
				if err != nil {
					continue
				}
				switch code := regexp_image_state.FindAllStringSubmatch(string(logindata), -1)[0][1]; code {
					case "65":
						fmt.Println("二维码已失效")
						return
					case "66":
						fmt.Println("二维码未失效，请扫描")
					case "67":
						fmt.Println("二维码正在验证..")
					case "0":
						fmt.Println("二维码验证成功")
						sig_link := ""
						if reg_sig := regexp.MustCompile(`ptuiCB\(\'0\',\'0\',\'([^\']+)\'`).FindAllStringSubmatch(string(logindata), -1); len(reg_sig) == 1 {
							sig_link = reg_sig[0][1]//获取扫描成功后的跳转地址
						} else {
							fmt.Println("Check Sig Err:")
							return
						}

						if _, err := r.Get(sig_link); err != nil {
							fmt.Println("Get Err:", err.Error())
							return
						}
						break validate_login
					default:
						fmt.Println("未知状态(" + code + ")")
						return
				}
			}

		//获取vpwebqq
		r.header["Referer"] = "http://s.web2.qq.com/proxy.html?v=20130916001&callback=1&id=1"
		vfwebqqdata, err := r.Get("http://s.web2.qq.com/api/getvfwebqq?ptwebqq=&clientid=53999199&psessionid=&t=" + r.GetTimestamp())
		if err != nil {
			log.Fatal(err)
		}
		sj, err := simplejson.NewJson(vfwebqqdata)
		if err != nil {
			log.Fatal(err)
		}
		if retcode, _ := sj.Get("retcode").Int(); retcode != 0 {
			return
		}
		vfwebqq, err := sj.Get("result").Get("vfwebqq").String()
		if err != nil {
			return
		}
		r.parameter["vfwebqq"] = vfwebqq
	}

	r.header["Referer"] = "http://d1.web2.qq.com/proxy.html?v=20151105001&callback=1&id=2"
	psessiondata, err := r.Post("http://d1.web2.qq.com/channel/login2", H{
		"r": "{\"ptwebqq\":\"\",\"clientid\":53999199,\"psessionid\":\"\",\"status\":\"online\"}",
	})

	if err != nil {
		return
	}
	sj, err := simplejson.NewJson(psessiondata)
	if err != nil {
		return
	}
	if retcode, _ := sj.Get("retcode").Int(); retcode != 0 {
		return
	}
	psessionid, err := sj.Get("result").Get("psessionid").String()
	if err != nil {
		return
	}
	r.parameter["psessionid"] = psessionid

	uid, err := sj.Get("result").Get("uin").Int()
	log.Printf("uid:%d",uid)
	r.uid = uid

	r.getOnline()

	r.getFriend()
	r.getGroup()
	r.getSelf()
	if r.onLogin != nil {
		r.onLogin(r)
	}
	if r.onMessage != nil {
		r.pollMessage()
	}
}

func (r *Robot) getOnline() {
		fmt.Printf("\n")
		log.Printf("获取在线...\n")
		r.header["Origin"] = "http://s.web2.qq.com"
		r.header["Referer"] = "http://d1.web2.qq.com/proxy.html?v=20151105001&callback=1&id=2"
		data, _ := r.Get("http://d1.web2.qq.com/channel/get_online_buddies2?vfwebqq="+r.parameter["vfwebqq"]+"&clientid=53999199&psessionid="+r.parameter["psessionid"]+"&t=" + r.GetTimestamp())
		log.Printf("self:%s", string(data))
}

func (r *Robot) getFriend() {
		fmt.Printf("\n")
		log.Printf("获取好友...\n")
		r.header["Origin"] = "http://s.web2.qq.com"
		r.header["Referer"] = "http://s.web2.qq.com/proxy.html?v=20130916001&callback=1&id=1"
		data, _ := r.Post("http://s.web2.qq.com/api/get_user_friends2", H{
			//"r": "{\"vfwebqq\":\""+r.parameter["vfwebqq"]+"\",\"hash\":\"0059006E00950026\"}",
			//"r": "{\"vfwebqq\":\""+r.parameter["vfwebqq"]+"\",\"hash\":\"0059006E00950026\"}",
			"vfwebqq":r.parameter["vfwebqq"],
			"hash":"0059006E00950026",
		})
		log.Printf("friends:%s", string(data))
}
func (r *Robot) getGroup() {
		fmt.Printf("\n")
		log.Printf("获取群...\n")
		r.header["Origin"] = "http://s.web2.qq.com"
		r.header["Referer"] = "http://s.web2.qq.com/proxy.html?v=20130916001&callback=1&id=1"
		data, _ := r.Post("http://s.web2.qq.com/api/get_group_name_list_mask2", H{
			//"r": "{\"vfwebqq\":\""+r.parameter["vfwebqq"]+"\",\"hash\":\"0059006E00950026\"}",
			//"r": "{\"vfwebqq\":\""+r.parameter["vfwebqq"]+"\",\"hash\":\"0059006E00950026\"}",
			"vfwebqq":r.parameter["vfwebqq"],
			"hash":"0059006E00950026",
		})
		log.Printf("groups:%s", string(data))
}
func (r *Robot) getSelf() {
		fmt.Printf("\n")
		log.Printf("获取个人信息...\n")
		r.header["Origin"] = "http://s.web2.qq.com"
		r.header["Referer"] = "http://s.web2.qq.com/proxy.html?v=20130916001&callback=1&id=1"
		data, _ := r.Post("http://s.web2.qq.com/api/get_self_info2?t=" + r.GetTimestamp(), H{
			//"r": "{\"vfwebqq\":\""+r.parameter["vfwebqq"]+"\",\"hash\":\"0059006E00950026\"}",
			//"r": "{\"vfwebqq\":\""+r.parameter["vfwebqq"]+"\",\"hash\":\"0059006E00950026\"}",
			"vfwebqq":r.parameter["vfwebqq"],
			"hash":"0059006E00950026",
		})
		log.Printf("self:%s", string(data))
}

/*
func hash(b int, i string) string {
	var a H
	for (s = 0; s < len(i); s++){
		a[s % 4] ^= i.charCodeAt(s);
	}
	var j = ["EC", "OK"], d = [];
	d[0] = b >> 24 & 255 ^ j[0].charCodeAt(0);
	d[1] = b >> 16 & 255 ^ j[0].charCodeAt(1);
	d[2] = b >> 8 & 255 ^ j[1].charCodeAt(0);
	d[3] = b & 255 ^ j[1].charCodeAt(1);
	j = [];
	for (s = 0; s < 8; s++)
		j[s] = s % 2 == 0 ? a[s >> 1] : d[s >> 1];
	a = ["0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F"];
	d = "";
	for (s = 0; s < j.length; s++)
		d += a[j[s] >> 4 & 15], d += a[j[s] & 15];
	return d;
}
*/

func (r *Robot) pollMessage() {
	for {
		fmt.Printf("\n")
		log.Printf("获取信息...\n")
		r.header["Origin"] = "http://d1.web2.qq.com"
		r.header["Referer"] = "http://d1.web2.qq.com/proxy.html?v=20151105001&callback=1&id=2"
		/*
		data, err := r.Post("http://d1.web2.qq.com/channel/poll2", H{
			"ptwebqq":    r.parameter["ptwebqq"],
			"clientid":   "53999199",
			"psessionid": r.parameter["psessionid"],
			"key":        "",
		})
		*/
		data, err := r.Post("http://d1.web2.qq.com/channel/poll2", H{
			"r": "{\"ptwebqq\":\""+r.parameter["ptwebqq"]+"\",\"clientid\":53999199,\"psessionid\":\""+r.parameter["psessionid"]+"\",\"status\":\"online\"}",
		})

		if err == nil {
			code := ParseMessage(r, data)
			if code == 103 {
				log.Println("登陆失败。请先在浏览器访问http://w.qq.com/扫码登录，然后退出。重新启动程序")
				//break
				r.getOnline()
			}
		}
	}
}

func ParseMessage(r *Robot, msg []byte) int {
	sj, err := simplejson.NewJson(msg)
	if err != nil {
		return -1
	}
	//获取消息结果
	retcode, err := sj.Get("retcode").Int()
	log.Printf("code：%d\n", retcode)
	if err != nil {
		return -1
	}
	if retcode != 0 {
		return retcode
	}
	//获取消息类型
	poll_type, err := sj.Get("result").GetIndex(0).Get("poll_type").String()
	if err != nil {
		return -1
	}
	if len(poll_type) == 0 {
		return -1
	}
	//获取消息内容和meta信息
	value := sj.Get("result").GetIndex(0).Get("value")
	//消息内容
	contentArr, err := value.Get("content").Array()
	if err != nil {
		return -1
	}
	atable := len(contentArr) > 2
	content := ""
	if atable {
		for i := 2; i <= len(contentArr); i++ {
			content = content + value.Get("content").GetIndex(i).MustString()
		}
	} else {
		content = value.Get("content").GetIndex(1).MustString()
	}
	fromUin := value.Get("from_uin").MustInt()
	sendUin := value.Get("send_uin").MustInt()
	msgId := value.Get("msg_id").MustInt()
	msgType := value.Get("msg_type").MustInt()
	sendTime := value.Get("time").MustInt()
	toUin := value.Get("to_uin").MustInt()

	message := &Message{
		PollType: poll_type,//message group_message discu_message
		Content:  content,
		FromUin:  fromUin,//来源 群聊id 组聊id 私聊id
		SendUin:  sendUin,//发送者，私聊为0，其他为发送着id
		MsgId:    msgId,
		MsgType:  msgType,// 1文字
		Time:     sendTime,
		ToUin:    toUin,//接受者id
		Atable:   atable,
	}
	log.Printf("message：%v\n", message)

	if sendUin == 0 && fromUin == r.uid {
		log.Printf("来自己的私聊\n")
		return -1
	}

	if sendUin > 0 && sendUin == r.uid {
		log.Printf("来自己的群聊或组聊\n")
		return -1
	}


	if r.onMessage != nil {
		r.onMessage(r, message)
	}
	return 0
}

func (r *Robot) SendToBuddy(toUin int, message string) error {
	r.sendMessage("to", toUin, message)
	return nil
}

func (r *Robot) SendToGroup(toUin int, message string) error {
	r.sendMessage("group_uin", toUin, message)
	return nil
}

func (r *Robot) SendToDiscuss(toUin int, message string) error {
	r.sendMessage("did", toUin, message)
	return nil
}

//msg_id加密算法
var msg_num int64 = time.Now().Unix() % 1E4 * 1E4

func (r *Robot) sendMessage(sendType string, toUin int, msg string) {
	msg_num++

	r.header["Content-Type"] = "application/x-www-form-urlencoded"
	r.header["Origin"] = "https://d1.web2.qq.com"
	r.header["Referer"] = "https://d1.web2.qq.com/cfproxy.html?v=20151105001&callback=1&id=2"
	send_data := `{"` + sendType + `":` + fmt.Sprintf("%d", toUin) + `,"content":"[\"` + msg + `\",[\"font\",{\"name\":\"宋体\",\"size\":10,\"style\":[0,0,0],\"color\":\"000000\"}]]","face":528,"clientid":53999199,"msg_id":` + fmt.Sprint(msg_num) + `,"psessionid":"` + r.parameter["psessionid"] + `"}`
	send_url := ""
	switch sendType {
	case "to":
		send_url = "http://d1.web2.qq.com/channel/send_buddy_msg2"
	case "group_uin":
		send_url = "http://d1.web2.qq.com/channel/send_qun_msg2"
	case "did":
		send_url = "http://d1.web2.qq.com/channel/send_discu_msg2"
	default:
		return
	}
	resp_send, err := r.Post(send_url, H{
		"r": send_data,
	})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	result, err := simplejson.NewJson([]byte(resp_send))
	if err == nil {
		if retcode, err := result.Get("retcode").Int(); err == nil && retcode == 100001 {
			r.sendMessage(sendType, toUin, msg)
		}
	}
}

func (r *Robot) Get(url string) ([]byte, error) {
	r.header["Content-Type"] = "utf-8"
	return r.Request("GET", url, nil)
}

func (r *Robot) Post(url string, param H) ([]byte, error) {
	r.header["Content-Type"] = "application/x-www-form-urlencoded; param=value"
	return r.Request("POST", url, param)
}

func (r *Robot) Request(method, posturl string, param H) ([]byte, error) {
	v := url.Values{}
	if param != nil {
		for key, value := range param {
			v.Set(key, value)
		}
	}
	body := ioutil.NopCloser(strings.NewReader(v.Encode()))
	req, err := http.NewRequest(method, posturl, body)
	if err != nil {
		return nil, err
	}

	r.header["User-Agent"] = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36"
	if r.header != nil {
		for key, value := range r.header {
			req.Header.Set(key, value)
		}
	}

	/*
	u, _ := url.Parse(posturl)
	cookies := r.client.Jar.Cookies(u)
	log.Printf("param:\n%v\n\nheader:\n%v\n\ncookies:\n%v\n\n",v,r.header,cookies)
	*/

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (r *Robot) GetCookie(requesturl, cookieName string) (value string) {
	u, err := url.Parse(requesturl)
	if err != nil {
		return
	}
	cookies := r.client.Jar.Cookies(u)
	for _, c := range cookies {
		if c.Name == cookieName {
			value = c.Value
			break
		}
	}
	return
}

func (r *Robot) GetToken(t string) string {
	e := 0
	data := []byte(t)
	n := len(data)
	for i := 0; n > i; i++ {
		e += (e << 5) + int(data[i])
	}
	return strconv.FormatInt(int64(2147483647&e), 10)
}

func (r *Robot) GetTimestamp() string {
	return strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
}
