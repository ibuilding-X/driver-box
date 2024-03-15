local json = require("json")

-- decode 请求数据解码
function decode(raw)
    local data = json.decode(raw)

    if string.find(data.path, "on") then
        local devices = {
            {
                ["device_sn"] = "sensor_1",
                ["values"] = {
                    {
                        ["name"] = "onOff",
                        ["value"] = 1
                    }
                }
            }
        }
        return json.encode(devices)
    end

    if string.find(data.path, "off") then
        local devices = {
            {
                ["device_sn"] = "sensor_1",
                ["values"] = {
                    {
                        ["name"] = "onOff",
                        ["value"] = 0
                    }
                }
            }
        }
        return json.encode(devices)
    end

    return "[]"
end