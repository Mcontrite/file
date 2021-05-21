package mq

import (
	"fmt"
	"reflect"
	"runtime"
)

var done chan bool

// StartConsume : 接收消息
func StartConsume(qName, cName string, callback func(msg []byte) bool) {
	fname := runtime.FuncForPC(reflect.ValueOf(callback).Pointer()).Name()
	fmt.Println("Start Consuming callback: ", fname, " if==nil: ", callback == nil)
	msgs, err := channel.Consume(
		qName,
		cName,
		true,  //自动应答
		false, // 非唯一的消费者
		false, // rabbitMQ只能设置为false
		false, // noWait, false表示会阻塞直到有消息过来
		nil)
	fmt.Println("Start Consuming, msgs: ", msgs, " if==nil: ", msgs == nil, " len(msgs)=", len(msgs))
	if err != nil {
		fmt.Println("Start RabbitMQ Consumer Error: ", err)
		return
	}
	done = make(chan bool)
	go func() {
		// 循环读取channel的数据
		fmt.Println("Start A Goroutine...")
		for d := range msgs {
			fmt.Println("Comsuming Messages 1")
			processErr := callback(d.Body)
			if processErr {
				// TODO: 将任务写入错误队列，待后续处理
				fmt.Println("Comsuming Messages 2")
			}
			fmt.Println("Comsuming Messages 3")
		}
	}()
	// 接收done的信号, 没有信息过来则会一直阻塞，避免该函数退出
	fmt.Println("Init RabbitMQ Consumer OK...")
	<-done
	// 关闭通道
	fmt.Println("Closing RabbitMQ Consumer...")
	channel.Close()
}

// StopConsume : 停止监听队列
func StopConsume() {
	done <- true
}
