package rabbitmq

import (
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// 🌟 1. 全局唯一的 TCP 连接
var GlobalConn *amqp.Connection

// 🌟 2. 项目启动时调用一次，修筑主干道
func InitRabbitMQ(mqUrl string) {
	var err error
	GlobalConn, err = amqp.Dial(mqUrl)
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

// amqp://用户名:密码@主机地址:端口号/VirtualHost
const MQURL = "amqp://admin:123456@127.0.0.1:5672/seckill"

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

func NewRabbitMQ(queueName string, exchange string, key string) *RabbitMQ {
	return &RabbitMQ{QueueName: queueName, Exchange: exchange, Key: key, Mqurl: MQURL}
}

// 断开各自channel
func (r *RabbitMQ) Destory() {
	if r.channel != nil {
		r.channel.Close()
	}
}

// 错误处理函数
func (r *RabbitMQ) failOnErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s:%s", message, err)
	}
}

// work模式

// work模式创建RabbitMQ实例，建立连接
func NewRabbitMQWork(queueName string) *RabbitMQ {
	if GlobalConn == nil {
		log.Fatal("RabbitMQ global connection not initialized!")
	}
	//创建RabbitMQ实例
	rabbitmq := NewRabbitMQ(queueName, "", "")
	rabbitmq.conn = GlobalConn
	var err error
	//从连接上开一条会话 → channel
	rabbitmq.channel, err = GlobalConn.Channel()
	rabbitmq.failOnErr(err, "failed to open a channel")
	return rabbitmq
}

// work模式下生产者发消息
func (r *RabbitMQ) PublishWork(message []byte) {
	//调用channel 发送消息到队列中
	err := r.channel.Publish(
		r.Exchange,
		r.QueueName,
		//如果为true，根据自身exchange类型和routekey规则无法找到符合条件的队列会把消息返还给发送者
		false,
		//如果为true，当exchange发送消息到队列后发现队列上没有消费者，则会把消息返还给发送者
		false,
		amqp.Publishing{
			ContentType:  "text/plain",
			Body:         message,
			DeliveryMode: amqp.Persistent,
		})
	if err != nil {
		fmt.Println(err)
	}
}

// 定义回调函数签名
type DoTaskFunc func(msg []byte) error

// work模式下消费者收消息，传入回调函数，接收消息执行回调函数，执行成功返回ack
func (r *RabbitMQ) ReceiveWork(taskFunc DoTaskFunc) {
	// work模式无需创建交换机，默认交换机在连上mq时就创建好了
	r.channel.Qos(1, 0, false) // 能者多劳

	dlxExchangeName := "seckill_dlx_exchange"
	dlxQueueName := "seckill_dead_queue"
	// 声明死信交换机（普通的 direct 即可）
	r.channel.ExchangeDeclare(dlxExchangeName, "direct", true, false, false, false, nil)
	// 声明死信队列
	r.channel.QueueDeclare(dlxQueueName, true, false, false, false, nil)
	// 绑定死信队列（我们让所有死信的 RoutingKey 都强制变成 "dead"）
	r.channel.QueueBind(dlxQueueName, "dead", dlxExchangeName, false, nil)

	//配置args作为队列的死信参数
	args := make(amqp.Table)
	// 指定死信交换机名字
	args["x-dead-letter-exchange"] = dlxExchangeName
	// 指定死信队列RoutingKey
	args["x-dead-letter-routing-key"] = "dead"

	q, _ := r.channel.QueueDeclare(r.QueueName, true, false, false, false, args)
	msgs, _ := r.channel.Consume(q.Name, "", false, false, false, false, nil)

	forever := make(chan bool)
	go func() {
		for d := range msgs {
			var err error
			maxRetry := 3 // 最大重试次数
			for i := range maxRetry {
				err = taskFunc(d.Body)

				if err == nil {
					break
				}
				log.Printf("⚠️ 订单处理失败 (第 %d/%d 次): %v", i+1, maxRetry, err)
				// 失败后等1s再试 (类似 Spring 的 BackOff)
				time.Sleep(1 * time.Second)
			}
			if err == nil {
				d.Ack(false)
			} else {
				//重试次数耗尽，发给死信队列，人工处理
				log.Printf("❌ %d 次重试全部失败,消息进入死信队列: %s", maxRetry, string(d.Body))
				d.Nack(false, false)
			}

		}
	}()
	log.Printf(" [*] 底层监听组件就绪...")
	<-forever
}
