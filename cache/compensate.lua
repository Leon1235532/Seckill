-- 补偿脚本: publish彻底失败时执行, 把秒杀Lua扣掉的东西精确还回去
-- KEYS[1]: 库存key (products:stock:%d)
-- KEYS[2]: 已购集合key (products:order:%d)
-- ARGV[1]: 用户ID
-- 两条命令焊成一个原子操作, 防止"还了库存但没删标记"的中间态
redis.call('INCR', KEYS[1])           -- ⇨ 原生命令: INCR products:stock:1  (库存还回1)
redis.call('SREM', KEYS[2], ARGV[1])  -- ⇨ 原生命令: SREM products:order:1 5 (划掉已购标记)
return 1