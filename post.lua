-- 初始化请求的 Method 和 Header
math.randomseed(os.time())
wrk.method = "POST"
wrk.headers["Content-Type"] = "application/json"

-- request 函数在 wrk 每次发请求前都会被调用
request = function()
   -- 动态生成 1 到 100000 之间的随机 user_id
   local user_id = math.random(1, 1000000)
   -- 拼装 JSON 请求体，product_id 固定为 1
   local body = '{"uid": ' .. user_id .. ', "pid": 1}'
   
   -- 发送请求
   return wrk.format(nil, nil, nil, body)
end