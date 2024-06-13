local json = require("json")

local HOLDING_REGISTER = "HOLDING_REGISTER"
local COIL = "COIL"
local DISCRETE_INPUT = "DISCRETE_INPUT"
local INPUT_REGISTER = "INPUT_REGISTER"

-- （勿动）模拟从机数据
local slaves = {

}

-- （勿动）初始化寄存器表
function initRegisters(length)
    local modbus_registers = {}
    for i = 1, length do
        modbus_registers[i] = 0
    end
    return modbus_registers
end

-- 初始化指定从机
function initSlave(slaveId, holdingRegister, coil, discreteInput, inputRegister)
    slaves[slaveId] = {
        [HOLDING_REGISTER] = initRegisters(holdingRegister),
        [COIL] = initRegisters(coil),
        [DISCRETE_INPUT] = initRegisters(discreteInput),
        [INPUT_REGISTER] = initRegisters(inputRegister)
    }

    -- 调用 mockWrite 方法初始化模拟数据
    -- Begin：以下需要开发者根据实际情况作修改


    -- End：以上需要开发者根据实际情况作修改

    return slaves[slaveId]
end


--模拟modbus读写
-- slaveId 从机id
-- primaryTable 寄存器类型：HOLDING_REGISTER,COIL,DISCRETE_INPUT,INPUT_REGISTER
-- address 寄存器地址
-- value 值，byte数组
function mockWrite(slaveId, primaryTable, address, value)
    if address < 1 or address > 65535 then
        error("Invalid register address")
    end
    -- 寻找从机
    slave = slaves[slaveId]
    if slaves[slaveId] == nil then
        slave = initSlave(slaveId, 65535, 65535, 65535, 65535);
    end

    tableData = slave[primaryTable]
    --从address开始填充数据value
    for i = 1, #value do
        tableData[address + i - 1] = value[i]
    end

    -- 对于读写点分离的情况，需要手动填写读点位数值
    -- Begin：以下需要开发者根据实际情况作修改

    -- End：以上需要开发者根据实际情况作修改
end

-- （勿动）模拟modbus读
function mockRead(slaveId, primaryTable, address, length)
    if address < 1 or address > 65535 then
        error("Invalid register address")
    end
    if length < 1 or length > 65535 then
        error("Invalid register length")
    end
    if address + length - 1 > 65535 then
        error("Invalid register length")
    end
    -- 寻找从机
    slave = slaves[slaveId]
    if slaves[slaveId] == nil then
        slave = initSlave(slaveId, 65535, 65535, 65535, 65535);
    end

    --从address开始读取length字节的数据
    tableData = slave[primaryTable]
    local data = {}
    for i = 1, length do
        local registerAddress = address + i - 1
        if registerAddress <= #tableData then
            table.insert(data, tableData[registerAddress])
        else
            error("Reading beyond the available registers")
        end
    end
    return json.encode(data)
end