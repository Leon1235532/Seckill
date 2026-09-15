<div align="center">

# Seckill

**基于 Go 的单机秒杀系统**

高并发下的库存扣减、削峰填谷与故障自愈。压测验证不超卖、不重复发货。

</div>

## 目录

- [About The Project](#about-the-project)
- [Built With](#built-with)
- [Architecture](#architecture)
- [核心设计](#核心设计)
- [Getting Started](#getting-started)
- [API](#api)
- [压测](#压测)
- [Roadmap](#roadmap)

## About The Project

秒杀的核心矛盾：瞬时流量远超数据库承载能力，同时库存扣减不允许任何差错——超卖是事故，重复发货也是事故。

项目的整体思路是把"能不能买到"的判定全部收敛到 Redis 侧：一段 Lua 脚本原子地完成时间窗校验、查重、查库存、扣减四个动作，判定成功才把订单消息丢进 RabbitMQ 异步落库。数据库自始至终不直接承受秒杀流量，只有消费者以稳定的速率写入。

几个实测数据：

- 查询接口 wrk 压测 2.2w+ QPS，压测全程 MySQL `Com_select` 计数只增加了 1——并发回源被 singleflight 合并成了唯一一次查库，缓存击穿无从谈起
- 秒杀接口 wrk `-t4 -c1000` 打出 2.9w QPS 洪峰、平均延迟 4.83ms：令牌桶拦截 28w 请求，放行流量经 Lua 原子判定后异步落库
- 压测结束后核账：订单数与扣减数一致，同一 uid 只有一条订单，没有超卖也没有重复

## Built With

<a href="https://skillicons.dev">
  <img src="https://skillicons.dev/icons?i=go,redis,mysql,rabbitmq,lua&perline=6" />
</a>

- **Gin** — HTTP 框架，限流中间件挂在 seckill 路由上
- **GORM** — MySQL ORM，软删除、AutoMigrate、唯一索引兜底
- **go-redis** — Redis 客户端，Lua 脚本经 EVALSHA 调用（只传哈希，不重传脚本）
- **RabbitMQ**（amqp091-go）— 异步削峰，work 模式多消费者
- **gobreaker** — 发布侧熔断器
- **singleflight**（golang.org/x/sync）— 合并同 key 并发回源
- **x/time/rate** — 令牌桶限流
- **wrk** — 压测，配套 Lua 脚本做状态码分布统计

## Architecture

秒杀请求的完整链路：

```
POST /seckill
    │
    ▼
令牌桶限流 ──拒绝──▶ 429
    │
    ▼
Redis Lua 脚本（单线程原子执行）
    │   时间窗 → 查重(已购Set) → 查库存 → DECR 扣减
    │
    ├── 1/2/3/4 ──▶ 库存不足 / 重复下单 / 未开始 / 已结束，直接返回
    │
    └── 0 扣减成功
         │
         ▼
    RabbitMQ（持久化消息）
         │ 发布失败 → 进程内重试 3 次
         │   └─ 仍失败 → 补偿脚本回滚库存与已购标记 → 返回 500
         │
         ▼
    消费者协程 × N（QoS=1，能者多劳）
         │
         ▼
    MySQL orders 表（(uid,pid) 唯一索引兜底防重）
         │ 消费失败 → 重试 3 次 → Nack(requeue=false)
         │                        → 死信队列，人工处理
         └─ Ack
```

熔断器挂在发布侧：MQ 不可用时先靠重试硬扛，连续 5 次失败跳闸，跳闸期间的请求在进入 Redis 之前就被 503 拦截——不让"库存已扣、消息发不出"的窟窿继续扩大；30 秒后半开放行一个请求探测，成功即恢复。

查询是独立的一条链路：`GET /checkpdt/:pid` 命中 info 缓存直接返回，未命中经 singleflight 合并回源 MySQL 并回填。

## 核心设计

**为什么用 Lua 而不是锁**

库存扣减方案前后迭代了四版，每一步都是压测逼出来的：

1. **Go 互斥锁版**：包级 `sync.Mutex` 包住"查库存 + 扣减 + 建单"整段流程。并发正确性没问题，但所有请求在应用层排队串行，wrk 实测吞吐约 400 QPS。
2. **MySQL 乐观锁版**：CAS 条件更新 `UPDATE ... SET stock = stock - 1 WHERE id = ? AND stock = ?`，以 `RowsAffected == 0` 判定抢购失败。第一版实现踩过一个经典坑：先建订单后扣库存，且 GORM 的 Update 匹配不到行时 err 为 nil，导致压测打出一堆"订单在、库存没扣"的幽灵订单——加上行数判断后才真正防住。但吞吐只到约 800 QPS：应用层并行了，最终那行 UPDATE 仍被 InnoDB 行锁串行化，每请求还有两次 DB 往返，瓶颈只是从 Go 的锁搬进了 DB 的行锁。
3. **Redis Lua 版**：判重、查库存、扣减在 Redis 单线程内合并为一个原子操作，判定路径完全不碰 MySQL——失败请求在 Redis 层就被挡掉，MySQL 只接收异步订单写入，wrk 实测 1w+ QPS。
4. **限流 + Lua + 异步落库（当前版）**：Lua 前加令牌桶把洪峰整形，扣减成功投递 MQ、订单异步落库，接口路径上不再有任何同步 DB 操作。wrk `-t4 -c1000` 实测 2.9w QPS、平均延迟 4.83ms。

四版演进的主线一句话：**把"判定"从磁盘挪进内存，把"落库"从同步变异步**。400 → 800 → 1w → 2.9w，每一步都在回答同一个问题——瓶颈在哪，就把那一层从请求路径上摘掉。

<details>
<summary>版本一：Go 互斥锁（伪代码）</summary>

```go
var mu sync.Mutex // 包级一把锁（第一版误写成函数内局部变量,各请求各锁各的,等于没锁）

func SaleHandler(c *gin.Context) {
    mu.Lock() // 锁包住"查 + 扣"整段,全程同步打 DB
    count := dao.QueryStock(pid)   // SELECT stock ...        (往返 1)
    if count > 0 {
        dao.InsertOrder(uid, pid)  // INSERT order + UPDATE stock-1 (往返 2、3)
    }
    mu.Unlock()
}
```

</details>

<details>
<summary>版本二：MySQL 乐观锁（伪代码）</summary>

```go
func SaleHandler(c *gin.Context) {
    stock := dao.QueryStock(pid)          // handler 先查库存,查到的值是 CAS 依据
    err := dao.InsertOrder(stock, uid, pid)
    // err == ErrStockChanged → 400 "被人抢了"
}

func InsertOrder(stock int, uid, pid uint) error {
    // ① 先扣:CAS 条件更新,WHERE 带上"我看到的库存值"
    res := DB.Model(&Product{}).
        Where("id = ? AND stock = ?", pid, stock).
        Update("stock", gorm.Expr("stock - ?", 1))
    if res.RowsAffected == 0 {   // 0 行 = 库存被人改过 = 抢购失败
        return ErrStockChanged   // 缺这一判断 CAS 等于白装(GORM 匹配不到行时 err 为 nil)
    }
    // ② 后建单(第一版顺序反了:先建单后扣库存,扣失败时订单已入库,压测打出一批"订单在、库存没扣"的幽灵订单)
    return DB.Create(&Order{UserID: uid, ProductID: pid}).Error
}
```

</details>

两版锁方案的结论：悲观锁和乐观锁都会把热点行钉死在数据库上，锁的代价随并发线性上涨；把判定挪进 Redis 等于直接使用其单线程模型的天然互斥。至于 SETNX 类分布式锁，需要额外处理超时、误删、续期，有原子操作可用时就不必引入。

**为什么订单异步落库**

秒杀用户的体验核心是"秒回"，而订单写入 MySQL 要走磁盘 IO 和索引维护，不该挡在响应路径上。Redis 扣减成功即返回，订单经 MQ 异步补写。代价是 MQ 的 at-least-once 语义可能重复投递，这里用 orders 表的 `(uid, pid)` 唯一索引兜底：重复消息插入时触发唯一键冲突被拒，天然幂等。

**发布侧可靠性：重试 → 补偿 → 熔断**

MQ 是这条链路里唯一可能"扣了库存却没生成订单"的环节，发布侧因此做了三层防御，层层递进：

1. **进程内重试**：发布失败按 attempt 重试 3 次（间隔 1s），每次 attempt 重新走 GetConn——连接已断则自动重拨，每条消息用独立 channel 发送。单次网络抖动在这一层就被吸收。
2. **原子补偿**：3 次全部耗尽时，执行补偿脚本把 Lua 已扣的库存 +1、用户移出已购集合——与扣减动作构成严格逆操作——然后返回 500。用户看到的失败是真实的，库存账是平的，重试是安全的。
3. **熔断兜底**：gobreaker 持续记账发布结果，连续 5 次失败跳闸 30 秒。跳闸期间新请求在进入 Redis 之前就被 503 拦截（此时没有扣库存，无需补偿），半开状态放行 1 个探子请求，成功即恢复。避免 MQ 长时间宕机时每个请求都白白重试 3 秒。

一个有意接受的边界：Publish 返回 nil 并不代表 broker 已落盘（那是 publisher confirm 的职责，列入 Roadmap）。在"响应丢失"的不确定态下，补偿可能多还 1 个库存——宁可账上多出一格，也不允许吞掉一单。

**缓存一致性**

单飞模式请求数据库拿到商品info后，同步放入缓存，如何尽量保证数据的真实性呢？

- 读：先查 Redis 的 products:info:{pid}，命中直接返回。没命中就把这个查询"合并"——同一时刻不管多少个请求在抢同一个 pid，只有 1 个真正去查 MySQL（singleflight，防缓存击穿），查完回填缓存，其他请求共享这份数据。
- 写（改/删接口）：先改 MySQL，然后删掉 info 缓存。下次有人读时自然重建。

两道防线保证缓存不至于太离谱：

1. 接口的增删改都会同步删缓存（第一道）
2. info 缓存只活 5 秒——绕过接口的改动（比如直接改库）没人帮你删缓存，靠 3 秒过期自动修正（第二道兜底）

一个有意接受的取舍：秒杀进行中，详情页展示的库存是过期的旧值，和 Redis 里正在被扣的真实库存对不上。但我们不去修它——因为"能不能抢到"永远以 Lua 脚本的判定为准，展示页只是给用户看看的，脏 3 秒无所谓。就像 12306：列表页显示有票，点进去照样可能告诉你没票；交易环节准确就行，展示环节允许撒谎。

## Getting Started

环境要求：Go 1.26+、MySQL 8、Redis 7、RabbitMQ。

```bash
git clone https://github.com/Leon1235532/Seckill.git
cd Seckill
go mod tidy
```

配置两个文件：

`config/config.ini`，各服务地址按本机实际修改：

```ini
port = 8080
release = false

[mysql]
host = 127.0.0.1
port = 3308
db = seckill

[redis]
host = 127.0.0.1
port = 6380

[rabbitmq]
host = 127.0.0.1
port = 5672
vhost = seckill
```

`config/.env`，敏感信息（不入库、不提交）：

```env
MYSQL_USER=root
MYSQL_PASSWORD=你的密码
RABBITMQ_USER=admin
RABBITMQ_PASSWORD=你的密码
```

RabbitMQ 需要预先创建 vhost 并授权：

```bash
rabbitmqctl add_vhost seckill
rabbitmqctl set_permissions -p seckill admin ".*" ".*" ".*"
```

MySQL 表由 AutoMigrate 自动创建，业务队列与死信队列由服务启动时自动声明，无额外初始化。

```bash
go run main.go
```

## API

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/addpdt` | 创建商品，同步预热 Redis（库存、起止时间） |
| POST | `/modify/:pid` | 修改商品，指针字段实现部分更新，先改 DB 再同步缓存 |
| DELETE | `/delete/:pid` | 软删除商品，同步清理所有相关缓存键 |
| GET | `/checkpdt/:pid` | 商品详情，info 缓存 + singleflight 回源 |
| POST | `/seckill` | 秒杀下单（挂令牌桶限流中间件） |

请求示例：

```bash
curl -X POST http://127.0.0.1:8080/addpdt \
  -H "Content-Type: application/json" \
  -d '{"name":"GPW","stock":100,"starttime":"2026-08-30T09:00:00+08:00","endtime":"2026-08-31T09:00:00+08:00"}'

curl -X POST http://127.0.0.1:8080/seckill \
  -H "Content-Type: application/json" \
  -d '{"pid":1,"uid":1}'
```

`/seckill` 的状态码语义：

| code | 含义 |
| --- | --- |
| 200 | 抢购成功，订单异步落库 |
| 400 | 库存不足 / 重复下单 / 未开始 / 已结束（msg 区分） |
| 429 | 触发限流 |
| 500 | MQ 异常，库存已补偿回滚，可安全重试 |
| 503 | 熔断跳闸中，稍后重试 |

## 压测

压测脚本 `post.lua`（wrk 的 Lua 扩展）：请求体随机生成 uid 模拟不同用户抢购，`response()` 回调按状态码分类统计，`done()` 阶段输出汇总。只看 wrk 自带的 Non-2xx 数是不够的——400 里"库存不足"和"重复下单"的占比决定了这次压测有没有意义。

```bash
wrk -t4 -c1000 -d30s -s post.lua http://127.0.0.1:8080/seckill
```

一轮实际输出（10s 短压，限流桶容量调小后）：

```text
Requests/sec:  29028.16      Latency(avg): 4.83ms
----------------------------------------------
200  OK                    :     100   成交
429  Too Many Requests     :  280329   令牌桶拦截
400  业务拒绝(含超时)       :   10937   Lua 时间窗/查重/库存不足
```

正确性核账看三件事：orders 表订单数 ≤ 初始库存（不超卖）、每个 uid 至多一条订单（不重复）、库存归零后 400 全部是"库存不足"（无重复扣减）。压测环境为本机 WSL2，绝对 QPS 受宿主机限制，一致性指标才是重点。

## Roadmap

单机版有意保留的边界：

- 熔断器是进程内状态，多实例部署时各跳各的，总故障容忍度随实例数线性放大
- 发布侧未启用 publisher confirm，broker 是否真正落盘依赖消息持久化属性
- 消费者数量固定为 10，未暴露配置

分布式版规划：

- [ ] docker-compose 多实例 + nginx 负载均衡，对比单实例压测数据
- [ ] gRPC + Protobuf 拆分商品 / 订单服务，替换进程内函数调用
- [ ] etcd 服务注册发现，客户端负载均衡
- [ ] 分布式锁（定时任务防并发场景），Redlock 可靠性分析

---

<div align="center">

[![GitHub](https://img.shields.io/badge/GitHub-Leon1235532-181717?style=flat&logo=github&logoColor=white)](https://github.com/Leon1235532)

</div>
