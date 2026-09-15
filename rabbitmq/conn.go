package rabbitmq

import (
	"fmt"
	"log"
	"sync"

	"github.com/Leon1235532/Seckill/setting"
	amqp "github.com/rabbitmq/amqp091-go"
)

var connMu sync.Mutex

// 🌟 1. 全局唯一的 TCP 连接
var GlobalConn *amqp.Connection

// amqp://用户名:密码@主机地址:端口号/VirtualHost
var MQURL string

// 🌟 2. 项目启动时调用一次，建立主干Tcp连接
func InitRabbitMQ(cfg *setting.RabbitMQConfig) {
	var err error
	MQURL = fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.VirtualHost)
	GlobalConn, err = amqp.Dial(MQURL)
	if err != nil {
		log.Fatalf("RabbitMQ global TCP connect failed!: %v", err)
	}
	log.Println("RabbitMQ global TCP connect succeed!")
}

// 🌟 3. 项目彻底退出时，拆除主干道
func CloseRabbitMQ() {
	if GlobalConn != nil {
		GlobalConn.Close()
		log.Println("RabbitMQ 全局TCP连接已关闭。")
	}
}

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	//队列名称
	QueueName string
	//交换机名称
	Exchange string
	//bind Key 名称
	Key string
	//连接信息
	Mqurl string
}

// GetConn 返回可用连接; 断了就重拨。锁防断线瞬间10个消费者+请求协程同时重拨互相覆盖
func GetConn() (*amqp.Connection, error) {
	connMu.Lock()
	defer connMu.Unlock()
	if GlobalConn != nil && !GlobalConn.IsClosed() {
		return GlobalConn, nil // 活着, 直接复用
	}
	conn, err := amqp.Dial(MQURL)
	if err != nil {
		return nil, err // 连接失败, 交给attempt判负
	}
	// 连接成功再赋给全局连接
	GlobalConn = conn
	log.Println("RabbitMQ connected/reconnected!")
	return GlobalConn, nil
}

// 断开各自channel
func (r *RabbitMQ) Destory() {
	if r.channel != nil {
		r.channel.Close()
	}
}
