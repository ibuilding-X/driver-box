local json = require("json")

function decode(deviceId, points)
    local returnPoints = {}
    for _, point in pairs(points) do
        --print("value: " .. point["value"])
        table.insert(returnPoints, {
            name = point["name"],
            value = point["value"],
        })
    end
    return json.encode(returnPoints)
end

function encode(deviceId, rw, points)
    return error("this device can not be encoded")
end