package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

// 参数配置
const (
	RPCUrl  = "" // 测试地址
	AuthUrl = ""
	// 测试之前获取
	UserName    = ""
	Password    = ""
	token       = ""
	max         = 1   // 并发次数
	concurrency = 1   // 控制 goroutine 并发量
	duration    = 900 // 秒
	gid         = ""
)

type CreateMeeting struct {
	Method string `json:"method"`
	Params struct {
		Gid      string `json:"gid"`
		Duration int    `json:"duration"`
	} `json:"params"`
}

type Limit struct {
	number  int
	channel chan struct{}
}

// Limit struct 初始化
func New(number int) *Limit {
	return &Limit{
		number:  number,
		channel: make(chan struct{}, number),
	}
}

// Run 方法：创建有限的 go f 函数的 goroutine
func (limit *Limit) Run(f func()) {
	limit.channel <- struct{}{}
	go func() {
		f()
		<-limit.channel
	}()
}

// WaitGroup 对象内部有一个计数器，从0开始
// 有三个方法：Add(), Done(), Wait() 用来控制计数器的数量
var wg = sync.WaitGroup{}

func main() {
	cnt := 0
	start := time.Now()
	limit := New(concurrency) // New Limit 控制并发量

	onceToken := GetToken()
	params := struct {
		Gid      string `json:"gid"`
		Duration int    `json:"duration"`
	}{Gid: gid, Duration: duration}
	postBody := &CreateMeeting{
		Method: "call.create",
		Params: params,
	}
	reqByte, _ := json.Marshal(postBody)

	for i := 0; i < max; i++ {
		wg.Add(1)
		value := i
		goFunc := func() {
			fmt.Printf("start func: %d\n", value)
			// 发送请求
			client := &http.Client{
				// 超时时间
				//Timeout: 20 * time.Second,
				// 跳过证书
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
			req, httpNewErr := http.NewRequest("POST", RPCUrl, bytes.NewReader(reqByte))
			if httpNewErr != nil {

				fmt.Printf("httpNewErr %f\n", httpNewErr)
			}
			req.Header.Add("Authorization", "Bearer "+onceToken)
			resp, cErr := client.Do(req)
			if cErr != nil {
				fmt.Printf("cErr %f\n", cErr)
			} else {
				bResp, rErr := ioutil.ReadAll(resp.Body)
				if rErr != nil {
					fmt.Printf("rErr %f\n", rErr)
				}
				result := &Resp{}
				if jsonErr := json.Unmarshal(bResp, result); jsonErr != nil {
					fmt.Printf("%s failed[json failed]", jsonErr)
				} else {
					if result.JsonError != nil {
						fmt.Printf("result.JsonError %v\n", *result.JsonError)
					} else {
						cnt++
						fmt.Printf("Resp Body: %v\n", string(bResp))
					}
				}
			}
			wg.Done()
		}
		limit.Run(goFunc)
	}

	//阻塞主程序，防止循环执行没结束程序退出
	wg.Wait()

	fmt.Printf("耗时: %fs\n", time.Now().Sub(start).Seconds())
	//fmt.Printf("执行失败次数： %v次\n", errCnt)
	fmt.Printf("执行成功次数： %v次\n", cnt)
}

func GetToken() string {
	client := &http.Client{
		Timeout: 20 * time.Second,
		//跳过证书
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	req, httpNewErr := http.NewRequest("GET", AuthUrl, nil)
	if httpNewErr != nil {
		fmt.Printf("httpNewErr %f\n", httpNewErr)
	}
	req.SetBasicAuth(UserName, Password)
	resp, cErr := client.Do(req)
	if cErr != nil {
		fmt.Printf("cErr %f\n", cErr)
	} else {
		bResp, rErr := ioutil.ReadAll(resp.Body)
		if rErr != nil {
			fmt.Printf("rErr %f\n", rErr)
		}
		result := &Resp{}
		if jsonErr := json.Unmarshal(bResp, result); jsonErr != nil {
			fmt.Printf("%s failed[json failed]", jsonErr)
			return "nil"
		}
		if result.JsonError != nil {
			fmt.Printf("result.JsonError", *result.JsonError)
		}
		fmt.Printf("Resp Body: %v\n", string(bResp))

		return result.Token
	}
	return "nil"
}

type JsonError struct {
	Code    json.Number `json:"code"`
	Message string      `json:"message"`
}

type Resp struct {
	*JsonError `json:"error,omitempty"`
	Token      string `json:"token"`
}
