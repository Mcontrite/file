package mq

import (
	"filestore-server/config"
	"fmt"

	"github.com/streadway/amqp"
)

var conn *amqp.Connection
var channel *amqp.Channel

// 如果异常关闭，会接收通知
var notifyClose chan *amqp.Error

func init() {
	// 是否开启异步转移功能，开启时才初始化rabbitMQ连接
	if !config.AsyncTransferEnable {
		fmt.Println("RabbitMQ Producer Not Start...")
		return
	}
	if initChannel() {
		err := channel.NotifyClose(notifyClose)
		if err != nil {
			fmt.Println("channel.NotifyClose Error: ", err)
		}
	}
	// 断线自动重连
	go func() {
		for {
			select {
			case msg := <-notifyClose:
				conn = nil
				channel = nil
				fmt.Printf("Init onNotifyChannelClosed: %+v\n", msg)
				initChannel()
			}
		}
	}()
	fmt.Println("Init RabbitMQ Producer OK...")
}

func initChannel() bool {
	if channel != nil {
		fmt.Println("Init Channel != nil")
		return true
	}

	conn, err := amqp.Dial(config.RabbitURL)
	if err != nil {
		fmt.Println("AMQP Dial Error: ", err)
		return false
	}

	channel, err = conn.Channel()
	if err != nil {
		fmt.Println("AMQP Conn Error: ", err)
		return false
	}
	fmt.Println("Init Channel == nil")
	return true
}

// Publish : 发布消息
func Publish(exchange, routingKey string, msg []byte) bool {
	if !initChannel() {
		fmt.Println("Publish !initChannel() false")
		return false
	}
	err := channel.Publish(
		exchange,
		routingKey,
		false, // 如果没有对应的queue, 就会丢弃这条消息
		false, //
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        msg})
	if err == nil {
		fmt.Println("Channel Publish True...")
		return true
	}
	fmt.Println("Channel Publish False...", err)
	return false
}
