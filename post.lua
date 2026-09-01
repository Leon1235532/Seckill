-- 调试标记：确认脚本已加载
io.write(">>> post.lua LOADED <<<\n")

status_counts = {}

function response(status, headers, body)
    status_counts[status] = (status_counts[status] or 0) + 1
end

function done(summary, latency, requests)
    io.write("\n========== HTTP Status Code Distribution ==========\n")
    
    if next(status_counts) == nil then
        io.write("  ⚠️  NO RESPONSES CAPTURED (response() never called)\n")
        io.write("  Total requests: " .. summary.requests .. "\n")
        io.write("  Non-2xx/3xx:    " .. summary.errors.status .. "\n")
    else
        local codes = {}
        for code in pairs(status_counts) do
            table.insert(codes, code)
        end
        table.sort(codes)

        local total = 0
        for _, code in ipairs(codes) do
            io.write(string.format("  HTTP %d : %d\n", code, status_counts[code]))
            total = total + status_counts[code]
        end
        io.write(string.format("  %-11s: %d\n", "TOTAL", total))
    end
    
    io.write("====================================================\n")
end

-- 初始化请求的 Method 和 Header
math.randomseed(os.time())
wrk.method = "POST"
wrk.headers["Content-Type"] = "application/json"

-- 全局数组，用于在 setup 阶段收集所有线程
local threads = {}

-- 1. setup 阶段：在所有线程启动前调用
function setup(thread)
    -- 将当前线程对象存入数组，方便最后汇总数据
    table.insert(threads, thread)
end

-- 2. init 阶段：每个线程启动时都会调用
function init(args)
    -- 给每个线程初始化独立的计数器（避免并发写冲突）
    status_200 = 0
    status_429 = 0
    status_other = 0
end

-- 3. request 阶段：每次发送请求前调用（你的原代码）
request = function()
    local user_id = math.random(1, 1000000)
    local body = '{"uid": ' .. user_id .. ', "pid": 1}'
    return wrk.format(nil, nil, nil, body)
end

-- 4. response 阶段：每次收到 HTTP 响应时调用
function response(status, headers, body)
    -- 根据状态码累加当前线程的计数器
    if status == 200 then
        status_200 = status_200 + 1
    elseif status == 429 then
        status_429 = status_429 + 1
    else
        status_other = status_other + 1
    end
end

-- 5. done 阶段：压测结束时调用，用于汇总并打印结果
function done(summary, latency, requests)
    local total_200 = 0
    local total_429 = 0
    local total_other = 0

    -- 遍历所有线程，拉取它们各自的统计数据并汇总
    for _, thread in ipairs(threads) do
        total_200 = total_200 + thread:get("status_200")
        total_429 = total_429 + thread:get("status_429")
        total_other = total_other + thread:get("status_other")
    end

    -- 打印炫酷的自定义战报
    print("\n==========================================")
    print("🚥 流量防线 (Token Bucket) 压测战报 🚥")
    print("==========================================")
    print(string.format("🟢 [200 OK] 成功放行: %d 次", total_200))
    print(string.format("🔴 [429 Too Many Requests] 被限流拦截: %d 次", total_429))
    if total_other > 0 then
        print(string.format("🟡 [其他状态码] 异常/报错: %d 次", total_other))
    end
    print("==========================================\n")
end
