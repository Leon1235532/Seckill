-- 变量说明：
-- KEYS[1]：商品库存的 Redis Key (比如 "seckill:stock:1001")
-- KEYS[2]：记录已购买用户的 Redis Set Key (比如 "seckill:orders:1001")
-- KEYS[3]：活动开始时间的毫秒时间戳 (比如 "products:start:1")
-- KEYS[4]：活动结束时间的毫秒时间戳
-- ARGV[1]：当前抢购的用户 ID

local now = redis.call('Time')
local ms = now[1] * 1000 + math.floor(now[2] / 1000)

if ms < tonumber(redis.call('GET',KEYS[3]) or '0') then
    return 3 --还没到开抢时间
end
if ms > tonumber(redis.call('GET',KEYS[4]) or '0') then
    return 4 --活动已结束
end
-- 1. 查重：判断用户是不是已经在抢购成功的集合里了
if (redis.call('SISMEMBER', KEYS[2], ARGV[1]) == 1) then
    return 2 -- 返回 2 代表重复下单，直接拦截
end

-- 2. 查库存：获取当前库存量
local stock = tonumber(redis.call('GET', KEYS[1]))

-- 3. 判断库存是否大于 0
if (stock == nil or stock <= 0) then
    return 1 -- 返回 1 代表库存不足，已经被抢光了
end

-- 4. 终极绝杀：扣库存 + 记录用户
redis.call('DECR', KEYS[1])           -- 纯内存原子减 1
redis.call('SADD', KEYS[2], ARGV[1])  -- 把用户 ID 塞进已购集合

return 0 -- 返回 0 代表抢购成功！
