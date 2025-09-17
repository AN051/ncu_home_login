package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"time"
)

type User struct {
	Phone          string    `json:"phone"`
	Code           string    `json:"code"`
	CodeExpireTime time.Time `json:"code_expire_time"`
	LastSendTime   time.Time `json:"last_send_time"`
	TodaySendCount int       `json:"today_send_count"`
	IsLoggedIn     bool      `json:"is_logged_in"`
}

var users = make(map[string]*User)
var dataFile = "data.json"

// 工具函数
func isValidPhone(phone string) bool {
	match, _ := regexp.MatchString(`^1[3-9]\d{9}$`, phone)
	return match
}

// 字母+数字混合验证码
func generateCode() string {
	const base = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 6)
	rand.Read(b)
	for i := range b {
		b[i] = base[b[i]%byte(len(base))]
	}
	return string(b)
}

func getUser(phone string) *User {
	if _, ok := users[phone]; !ok {
		users[phone] = &User{Phone: phone}
	}
	return users[phone]
}

func canSendCode(user *User) bool {
	now := time.Now()
	if now.Sub(user.LastSendTime) < 60*time.Second {
		return false
	}
	if user.TodaySendCount >= 5 && user.LastSendTime.Day() == now.Day() {
		return false
	}
	return true
}

func sendCode(user *User) {
	now := time.Now()
	user.Code = generateCode()
	user.CodeExpireTime = now.Add(5 * time.Minute)
	user.LastSendTime = now
	if user.LastSendTime.Day() != now.Day() {
		user.TodaySendCount = 0
	}
	user.TodaySendCount++
	user.IsLoggedIn = false
}

func login(user *User, code string) bool {
	if user.Code != code || time.Now().After(user.CodeExpireTime) {
		return false
	}
	user.IsLoggedIn = true
	user.Code = ""
	return true
}

func loadData() {
	file, err := os.ReadFile(dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		fmt.Println("读取 data.json 出错：", err)
		return
	}
	if len(file) == 0 {
		return
	}
	var list []*User
	if err = json.Unmarshal(file, &list); err != nil {
		fmt.Println("解析 data.json 出错：", err)
		return
	}
	for _, u := range list {
		users[u.Phone] = u
	}
}

func saveData() {
	list := make([]*User, 0, len(users))
	for _, u := range users {
		list = append(list, u)
	}
	data, _ := json.MarshalIndent(list, "", "  ")
	os.WriteFile(dataFile, data, 0644)
}

// 控制台交互 main
func main() {
	loadData()
	defer saveData()

	fmt.Print("请输入手机号：")
	var phone string
	fmt.Scanln(&phone)

	if !isValidPhone(phone) {
		fmt.Println("手机号格式错误")
		return
	}
	user := getUser(phone)

	for {
		fmt.Println("1：输入验证码进行登录\n2：获取验证码\n0：退出")
		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 0:
			fmt.Println("已退出")
			return
		case 1:
			fmt.Print("请输入验证码：")
			fmt.Scanln() // 清空缓冲区
			var code string
			fmt.Scanln(&code) // 现在随便输 1/2/0 都不会被菜单抢跑
			if login(user, code) {
				fmt.Println("登录成功")
				return
			}
			fmt.Println("登录失败：无效验证码或已过期")
		case 2:
			if canSendCode(user) {
				sendCode(user)
				fmt.Println("验证码已发送：", user.Code)
			} else {
				fmt.Println("无法发送：60秒内重复或今日已达5次上限")
			}
		default:
			fmt.Println("未知操作")
		}
	}
}
