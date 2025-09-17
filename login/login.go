package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"time"
)

type User struct { // 定义用户结构体
	Phone          string    `json:"phone"`
	Code           string    `json:"code"`
	CodeExpireTime time.Time `json:"code_expire_time"`
	LastSendTime   time.Time `json:"last_send_time"`
	TodaySendCount int       `json:"today_send_count"`
	IsLoggedIn     bool      `json:"is_logged_in"`
}

var users = make(map[string]*User)
var dataFile = "data.json"

/* ---------- HTTP 接口(AI改) ---------- */
type R map[string]any // 偷懒写 JSON 回复

func handleSendCode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // 允许前端跨域

	phone := r.URL.Query().Get("phone")
	if !isValidPhone(phone) {
		json.NewEncoder(w).Encode(R{"ok": false, "msg": "手机号格式错误"})
		return
	}
	user := getUser(phone)
	if !canSendCode(user) {
		json.NewEncoder(w).Encode(R{"ok": false, "msg": "60秒内重复或今日已达5次上限"})
		return
	}
	sendCode(user)
	saveData()
	json.NewEncoder(w).Encode(R{"ok": true, "code": user.Code}) // 调试用，生产可去掉
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	phone := r.URL.Query().Get("phone")
	code := r.URL.Query().Get("code")
	if !isValidPhone(phone) {
		json.NewEncoder(w).Encode(R{"ok": false, "msg": "手机号格式错误"})
		return
	}
	user := getUser(phone)
	if login(user, code) {
		saveData()
		json.NewEncoder(w).Encode(R{"ok": true, "msg": "登录成功"})
	} else {
		json.NewEncoder(w).Encode(R{"ok": false, "msg": "无效验证码或已过期"})
	}
}

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
			var code string
			fmt.Scanln(&code)
			if login(user, code) {
				fmt.Println("登录成功")
				return
			} else {
				fmt.Println("登录失败：无效验证码或已过期")
			}
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

func isValidPhone(phone string) bool {
	match, _ := regexp.MatchString(`^1[3-9]\d{9}$`, phone)
	return match
}

// 生成验证码(AI修改验证格式)
func generateCode() string {
	const base = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ" // 把字母也加进来
	b := make([]byte, 6)                                // 准备 6 个空位置
	rand.Read(b)                                        // 随机填满
	for i := range b {
		b[i] = base[b[i]%byte(len(base))] // 把随机值映射到 base 里的一位
	}
	return string(b) // 拼成字符串
}

func getUser(phone string) *User {
	if _, exists := users[phone]; !exists {
		users[phone] = &User{Phone: phone}
	}
	return users[phone]
}

func canSendCode(user *User) bool {
	now := time.Now()
	if now.Sub(user.LastSendTime) < 60*time.Second { // 60秒内不能重复发送
		return false
	}
	if user.TodaySendCount >= 5 && user.LastSendTime.Day() == now.Day() { // 今日已达5次上限
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

func loadData() { //AI修改验证格式
	file, err := os.ReadFile(dataFile)
	if err != nil {
		// 文件不存在不算错，第一次运行本来就没有
		if os.IsNotExist(err) {
			return
		}
		// 真正异常才打印
		fmt.Println("读取 data.json 出错：", err)
		return
	}

	// 如果文件内容为空或格式不对，也会报错
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

func saveData() { //AI辅助修改，已加密，data不显示code
	list := make([]*User, 0, len(users))
	for _, u := range users { //装用户
		list = append(list, u)
	}
	data, _ := json.MarshalIndent(list, "", "  ") //格式化json
	os.WriteFile(dataFile, data, 0644)            //文件权限其他人只读
}

