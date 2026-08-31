wrk.method = "GET"
wrk.headers["Accept"] = "application/json"

request = function()
   -- path 传 nil，自动使用命令行的 /checkpdt/1
   return wrk.format(nil, nil, nil, nil)
end