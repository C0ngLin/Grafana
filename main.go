package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var timeValue time.Time
var myMap map[string]int
var executed bool

func handler(w http.ResponseWriter, r *http.Request) {
	// 读取请求的 body 内容
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}

	// 打印请求内容
	fmt.Println(string(body))

	// 将告警数据转换为企业微信消息
	message, err := translateAlertToWeCom(body)
	if err != nil {
		http.Error(w, "Error translating alert data", http.StatusInternalServerError)
		return
	}

	// 发送消息到企业微信
	if err := sendToWeCom(message); err != nil {
		http.Error(w, "Error sending message to WeCom", http.StatusInternalServerError)
		return
	}

	// 返回响应
	fmt.Fprintf(w, "Received request and sent message to WeCom")
}

// 初始化 myMap
func initMap() {
	myMap = make(map[string]int)
}

// 正则筛选数字
func extractNumbers(input string) string {
	// 定义匹配数字的正则表达式
	re := regexp.MustCompile(`\d+`)

	// 使用正则表达式提取数字
	matches := re.FindAllString(input, -1)

	// 将匹配到的数字组合成一个字符串
	result := ""
	for _, match := range matches {
		result += match
	}

	return result
}

// 根据-拆分模块的接口
func splitInterface(input interface{}, separator string) (environment, platform, module, api string, err error) {
	// 将 interface{} 转换为字符串
	inputStr, ok := input.(string)
	if !ok {
		err = fmt.Errorf("input is not a string")
		return
	}

	// 拆分字符串
	parts := strings.Split(inputStr, separator)

	// 检查是否有足够的部分
	if len(parts) != 4 {
		err = fmt.Errorf("invalid input format")
		return
	}

	// 分配各部分的值
	environment = parts[0]
	platform = parts[1]
	module = parts[2]
	api = parts[3]

	return
}

func translateAlertToWeCom(alertData []byte) (string, error) {

	// 解析接收到的 JSON 数据
	var alert map[string]interface{}
	if err := json.Unmarshal(alertData, &alert); err != nil {
		return "", err
	}

	// 获取 alerts 数组
	alerts, ok := alert["alerts"].([]interface{})
	if !ok || len(alerts) == 0 {
		return "", errors.New("找不到 alerts 数组或数组为空")
	}

	// 获取第一个 alert 元素
	firstAlert, ok := alerts[0].(map[string]interface{})
	if !ok {
		return "", errors.New("找不到 alert 对象")
	}

	// 获取 startsAt 字段
	startsAt, ok := firstAlert["startsAt"].(string)
	if !ok {
		return "", errors.New("找不到 startsAt 字段")
	}

	startsAtTime, err := time.Parse(time.RFC3339Nano, startsAt)
	if err != nil {
		return "", err
	}

	startsAtFormatted := startsAtTime.Format("2006-01-02 15:04:05")

	// 获取 annotations 字段
	annotations, ok := firstAlert["annotations"].(map[string]interface{})
	if !ok {
		return "", errors.New("找不到 annotations 字段")
	}

	//statusValue, ok := alert["status"].(string)
	//if !ok {
	//	fmt.Println("Error getting status value")
	//	return "", err
	//}

	// 获取 description 和 runbook_url
	//description := annotations["description"]
	runbookURL := annotations["runbook_url"]
	//AlertLevel := annotations["AlertLevel"]
	//AlarmSource := annotations["AlarmSource"]
	//AlarmType := annotations["AlarmType"]
	TriggeringConditions := annotations["TriggeringConditions"]
	ErrorCodeMeaning := annotations["ErrorCodeMeaning"]
	//NumberOfErrors := 0
	//AlarmStatus := statusValue
	AffectedComponents := annotations["AffectedComponents"]
	//Threshold := annotations["Threshold"]
	//HandlingSuggestions1 := annotations["HandlingSuggestions1"]
	//HandlingSuggestions2 := annotations["HandlingSuggestions2"]

	// 匿名函数，用于执行一次性的代码
	executeOnce := func() {
		if executed {
			return
		}
		// 在这里放置你想要执行一次的代码
		timeValue = time.Now()
		initMap()

		// 设置标志表示已经执行过
		executed = true
	}

	// 初始化 myMap

	executeOnce()
	currentTime := time.Now()

	// 获取 AffectedComponents 字段
	affectedComponentsString, ok := annotations["AffectedComponents"].(string)
	if !ok {
		return "", errors.New("找不到或 AffectedComponents 不是字符串类型")
	}

	currentDate := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC)
	anotherDate := time.Date(timeValue.Year(), timeValue.Month(), timeValue.Day(), 0, 0, 0, 0, time.UTC)
	// 如果当前时间和保存的时间是同一天，则添加计数
	if currentDate.Equal(anotherDate) {
		fmt.Printf("是同一天\n")
		if value, exists := myMap[affectedComponentsString]; exists {
			fmt.Printf("元素存在\n")
			fmt.Printf("Key %s exists. Value: %v\n", affectedComponentsString, value)
			myMap[affectedComponentsString] = myMap[affectedComponentsString] + 1
		} else {
			fmt.Println("元素不存在\n")
			fmt.Println(myMap[affectedComponentsString])
			fmt.Println(myMap)
			myMap[affectedComponentsString] = 1
		}
	} else {
		fmt.Printf("不是同一天\n")
		//如果当前时间和保存的时间不是同一天，则重置计数
		myMap = make(map[string]int)
		myMap[affectedComponentsString] = 1
	}

	timeValue = currentTime

	// 构建企业微信消息内容
	//message := fmt.Sprintf("自动告警通知\n警报级别: %s\n告警时间: %s\n告警来源: %s\n---\n告警详情:\n1. 告警类型:%s\n2. 描述: %s\n3. 触发条件: %s\n4. 仪表盘地址:  %s\n5. 错误码含义:  %s\n6. 今日报错次数:  %d\n---\n告警状态: %s\n受影响组件: %s\n阈值: %s\n---\n处理建议:\n1. %s\n2. %s\n---\n请尽快处理此告警，谢谢！",
	//AlertLevel, startsAtFormatted, AlarmSource, AlarmType, description, TriggeringConditions,
	//runbookURL, ErrorCodeMeaning, myMap[affectedComponentsString], AlarmStatus, AffectedComponents, Threshold,
	//HandlingSuggestions1, HandlingSuggestions2)
	// 尝试将 TriggeringConditions 转换为 string
	if str, ok := TriggeringConditions.(string); ok {
		// 转换成功，可以在这里使用 str
		fmt.Println("TriggeringConditions as string:", str)
		// 使用方法拆分字符串
		env, plat, mod, api, err := splitInterface(AffectedComponents, "-")
		if err != nil {
			fmt.Println("Error:", err)
			return "", nil
		}
		TriggeringConditions = extractNumbers(str)
		message := fmt.Sprintf("【平台接口报错同步】\n1、环境-对应端：%s\n2、模块-接口：%s\n3、报错code码：%s\n4、错误码含义：%s\n5、报错时间：%s\n6、今日报错次数：%d\n7、仪表盘地址：%s\n",
			env+"-"+plat, mod+"-"+api, TriggeringConditions, ErrorCodeMeaning, startsAtFormatted, myMap[affectedComponentsString], runbookURL)
		// 构建企业微信消息体
		messageData := map[string]interface{}{
			"msgtype": "text",
			"text": map[string]interface{}{
				"content": message,
			},
		}

		// 将消息体转为 JSON
		messageJSON, err := json.Marshal(messageData)
		if err != nil {
			return "", err
		}

		return string(messageJSON), nil
	} else {
		// 转换失败，处理错误
		fmt.Println("TriggeringConditions cannot be converted to string.")
	}

	// 构建企业微信消息体
	messageData := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]interface{}{
			"content": "错误码解析错误",
		},
	}

	// 将消息体转为 JSON
	messageJSON, err := json.Marshal(messageData)
	if err != nil {
		return "", err
	}

	return string(messageJSON), nil
}

func sendToWeCom(message string) error {
	wecomURL := "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=f9cd0dff-832b-41a2-93ec-c6b440b5c8c2"

	req, err := http.NewRequest("POST", wecomURL, bytes.NewBuffer([]byte(message)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to send message. Status code: %d", resp.StatusCode)
	}

	return nil
}

func main() {
	// 设置路由和处理函数
	http.HandleFunc("/", handler)

	// 启动 HTTP 服务器
	fmt.Println("Server is running on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		return
	}
}
