local json = require("json")

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

function decode(deviceId, points)
    local returnPoints = {}
    for _, point in pairs(points) do
        table.insert(returnPoints, {
            name = point["name"],
            value = point["value"],
        })
    end
    return json.encode(returnPoints)
end

function encode(deviceId, rw, points)
    local returnPoints = {}
    for _, point in pairs(points) do
        table.insert(returnPoints, {
            name = point["name"],
            value = point["value"],
        })
    end
    return json.encode(returnPoints)
end