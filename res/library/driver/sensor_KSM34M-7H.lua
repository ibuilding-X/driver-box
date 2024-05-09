local json = require("json")

local encodeParam = {

}

-- 格式化数字，最多保留两位小数
function format_number(num)
    v = math.floor(num)
    if num == v then
        return v
    end
    local formatted = string.format("%.2f", num)
    formatted = string.gsub(formatted, "%.?0+$", "")
    return formatted
end

-- decode 请求数据解码
function decode(deviceId, points)
    local returnPoints = {}
    for _, point in pairs(points) do
        --print("value: " .. point["value"])
        if point["name"] == 'hcho' then
            table.insert(returnPoints, {
                name = 'hcho',
                value = format_number((point["value"] / 1000 * 30.03) / 24.45),
            })
        elseif point["name"] == 'tvoc' then
            table.insert(returnPoints, {
                name = 'tvoc',
                value = format_number(point["value"] / 1000),
            })
        elseif point["name"] == 'temperature' or point["name"] == 'humidity' then
            table.insert(returnPoints, {
                name = point["name"],
                value = format_number(point["value"] * 0.1),
            })
        else
            table.insert(returnPoints, {
                name = point["name"],
                value = point["value"],
            })
        end
    end
    return json.encode(returnPoints)
end

function encode(deviceId, model, points)

    if not deviceId then
        print("deviceId not found")
        return "[]"
    end
    if not model then
        print("model not found")
        return "[]"
    end
    print("deviceId:" .. deviceId .. " model:" .. model)

    local returnPoints = {}
    for _, v in pairs(points) do
        for i, j in pairs(v) do
            local keyStr = tostring(i)
            local valueStr = tostring(j)
            print(keyStr .. ":" .. valueStr)
        end

    end
    if true then
        return "[]"
    end
    -- 空开需要先解锁，才可进行开合闸
    local data = json.decode(points)
    if string.find(data["modelName"], "circuit_breaker") then
        --print(point)
        -- 修改 ready => command
        if data["pointName"] == "ready" then
            data["pointName"] = "command"
            if tonumber(data.value) == 0 then
                data["value"] = 7
            elseif tonumber(data.value) == 1 then
                data["value"] = 6
            end
        end
        -- 开合闸之前自动执行解锁
        if data["pointName"] == "command" and (data["value"] == 6 or data["value"] == 7) then
            data["preOp"] = {
                {
                    ["pointName"] = "command",
                    ["value"] = 2
                }
            }
        end
    elseif data["pointName"] == "tempSetting" then
        data["value"] = data["value"] * 10
    end
    local res = json.encode(data)
    return res
end