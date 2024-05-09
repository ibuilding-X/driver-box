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

-- decode 美控7合1环境传感器
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
    return returnPoints
end

function encode(deviceId, model, points)
    return error("this device can not be encoded")
end