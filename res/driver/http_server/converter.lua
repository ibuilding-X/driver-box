local json = require("json")

-- decode 请求数据解码
-- curl -X POST -H "Content-Type: application/json" -d '{"id":"swtich-2","onOff":1}' http://127.0.0.1:8888/report
function decode(raw)
    local data = json.decode(raw)
    local body= json.decode(data.body)
    if data.method == "POST" and data.path == "/report" then
        local devices = {
            {
                ["id"] = body["id"], -- 设备ID
                ["values"] = {
                    { ["name"] = "onOff", ["value"] = body["onOff"] }, -- 点位解析
                }
            }
        }
        return json.encode(devices)
    else
        print("request error")
        return "[]"
    end
end