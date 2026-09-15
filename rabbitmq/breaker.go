package rabbitmq

import (
	"log"
	"time"

	"github.com/sony/gobreaker"
)

var MQBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
	Name:        "RabbitMQ-Breaker",
	MaxRequests: 1,                // HalfOpen 状态只放 1 个探子请求
	Timeout:     30 * time.Second, // 跳闸后的计时器时间
	ReadyToTrip: func(counts gobreaker.Counts) bool {
		return counts.ConsecutiveFailures >= 5 // 连续失败5次才跳闸
	},
	// 状态变化时的回调函数
	OnStateChange: func(name string, from, to gobreaker.State) {
		log.Printf("MQ circuit breaker: %s -> %s", from, to)
	},
})

// 熔断器状态是否为跳闸 Open
func MQOpen() bool {
	return MQBreaker.State() == gobreaker.StateOpen
}

func PublishWithBreaker(queueName string, body []byte, attempts int) error {
	// Execute函数内部初始化状态为Closed，会调用传入的匿名函数，成功则计算器清零，失败则计数器+1
	// 达到失败阈值(ReadyToTrip的规则)时，状态转为Open，Open状态时，匿名函数不会执行，直接返回内部错误ErrOpenState/ErrTooManyRequests
	// Open状态的计时器到期时，状态转为HalfOpen，放行一个请求执行匿名函数，成功则转Closed、失败切回Open，计时器恢复
	_, err := MQBreaker.Execute(func() (interface{}, error) {
		return nil, PublishWithRetry(queueName, body, attempts)
	})
	return err
}
