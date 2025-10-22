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
	LastSendDate   string    `json:"last_send_date"` // 新增：用于判断日期
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
	for i := range b {
		b[i] = base[rand.Intn(len(base))]
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
	today := now.Format("2006-01-02")
	
	// 新的一天，重置次数
	if user.LastSendDate != today {
		user.TodaySendCount = 0
		user.LastSendDate = today
	}
	
	// 60秒内重复发送
	if now.Sub(user.LastSendTime) < 60*time.Second {
		return false
	}
	
	// 今日已达5次上限
	if user.TodaySendCount >= 5 {
		return false
	}
	
	return true
}

func sendCode(user *User) {
	now := time.Now()
	user.Code = generateCode()
	user.CodeExpireTime = now.Add(5 * time.Minute)
	user.LastSendTime = now
	user.LastSendDate = now.Format("2006-01-02")
	user.TodaySendCount++
	user.IsLoggedIn = false
}

func login(user *User, code string) bool {
	if user.Code == "" || user.Code != code {
		return false
	}
	if time.Now().After(user.CodeExpireTime) {
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
		fmt.Println("解析 data.json 出错（文件可能损坏）：", err)
		// 备份损坏的文件
		os.Rename(dataFile, dataFile+".backup")
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
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		fmt.Println("保存数据出错：", err)
		return
	}
	if err := os.WriteFile(dataFile, data, 0644); err != nil {
		fmt.Println("写入文件出错：", err)
	}
}

// 控制台交互 main
func main() {
	rand.Seed(time.Now().UnixNano())
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
		fmt.Println("\n1：输入验证码进行登录")
		fmt.Println("2：获取验证码")
		fmt.Println("0：退出")
		fmt.Print("请选择操作：")
		
		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 0:
			fmt.Println("已退出")
			return
			
		case 1:
			fmt.Print("请输入验证码：")
			var code string
			fmt.Scanln(&code)
			
			if login(user, code) {
				fmt.Println("✓ 登录成功")
				return
			}
			fmt.Println("✗ 登录失败：验证码无效或已过期")
			
		case 2:
			if canSendCode(user) {
				sendCode(user)
				fmt.Printf("✓ 验证码已发送：%s（5分钟内有效）\n", user.Code)
				fmt.Printf("今日剩余次数：%d/5\n", 5-user.TodaySendCount)
			} else {
				now := time.Now()
				if now.Sub(user.LastSendTime) < 60*time.Second {
					wait := 60 - int(now.Sub(user.LastSendTime).Seconds())
					fmt.Printf("✗ 发送过于频繁，请等待 %d 秒后重试\n", wait)
				} else {
					fmt.Println("✗ 今日发送次数已达上限（5次）")
				}
			}
			
		default:
			fmt.Println("✗ 无效操作，请输入 0、1 或 2")
		}
	}
}
