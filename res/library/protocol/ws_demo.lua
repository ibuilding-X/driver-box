local json = require("json")

-- ws示例：{"id":"ws-swtich-2","points":[{"name":"onOff","value":"2"}]}
function decode(raw)
    local data = json.decode(raw)
    --for k, v in pairs(data) do
    --    print(k, v)
    --end
    if data["event"] ~= "read" then
        return "[]"
    end
    -- 打印 data
    print("data:" .. data["payload"])
    local payload = json.decode(data["payload"])

    local device = {
        ["id"] = payload["id"],
        ["values"] = {
        },
    }
    for _, point in pairs(payload["points"]) do
        --print("value: " .. point["value"])
        table.insert(device["values"], {
            ["name"] = point["name"],
            ["value"] = point["value"],
        })
    end

    return json.encode({ device })
end

function encode(deviceId, rw, points)
    return error("this device can not be encoded")
end