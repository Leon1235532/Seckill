package rabbitmq

import (
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var connMu sync.Mutex

// 🌟 1. 全局唯一的 TCP 连接
var GlobalConn *amqp.Connection

// 🌟 2. 项目启动时调用一次，建立主干Tcp连接
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

func NewRabbitMQ(queueName string, exchange string, key string) *RabbitMQ {
	return &RabbitMQ{QueueName: queueName, Exchange: exchange, Key: key, Mqurl: MQURL}
}

// 断开各自channel
func (r *RabbitMQ) Destory() {
	if r.channel != nil {
		r.channel.Close()
	}
}

// work模式

// work模式创建RabbitMQ实例，建立连接
func NewRabbitMQWork(queueName string) (*RabbitMQ, error) {
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	rabbitmq := NewRabbitMQ(queueName, "", "")
	rabbitmq.conn = conn
	rabbitmq.channel, err = conn.Channel()
	if err != nil {
		return nil, err
	}
	return rabbitmq, nil
}

// work模式下生产者发消息
func (r *RabbitMQ) PublishWork(message []byte) error {
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
		return err
	}
	return nil
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

	q, err := r.channel.QueueDeclare(r.QueueName, true, false, false, false, args)
	if err != nil {
		log.Fatalf("QueueDeclare %s failed: %v (check old queue params in mgmt console, delete seckill_queue and restart)", r.QueueName, err)
	}
	msgs, err := r.channel.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Consume %s failed: %v", q.Name, err)
	}

	for d := range msgs {
		var err error
		maxRetry := 3 // 最大重试次数
		for i := 1; i <= maxRetry; i++ {
			err = taskFunc(d.Body)

			if err == nil {
				break
			}
			log.Printf("⚠️ 订单处理失败 (第 %d/%d 次): %v", i, maxRetry, err)
			// 失败后等1s再试 (类似 Spring 的 BackOff)
			if i < maxRetry {
				time.Sleep(1 * time.Second)
			}
		}

		if err == nil {
			d.Ack(false)
		} else {
			//重试次数耗尽，发给死信队列，人工处理
			log.Printf("❌ %d 次重试全部失败,消息进入死信队列: %s", maxRetry, string(d.Body))
			d.Nack(false, false)
		}

	}
}

// 生产端重试: 每次attempt重新拿连接(断了GetConn里自动重拨), 全败返回最后的err
func PublishWithRetry(queueName string, body []byte, attempts int) error {
	var lastErr error
	for i := 1; i <= attempts; i++ {
		rmq, err := NewRabbitMQWork(queueName)
		if err != nil {
			lastErr = err
		} else {
			lastErr = rmq.PublishWork(body)
			rmq.Destory()
			if lastErr == nil {
				return nil
			}
		}
		log.Printf("publish attempt %d/%d failed: %v", i, attempts, lastErr)
		if i < attempts {
			time.Sleep(1 * time.Second) // ponytail: 固定退避, 空转天花板
		}
	}
	return lastErr
}
