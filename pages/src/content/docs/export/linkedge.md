---
title: 场景联动
description: A guide in my new Starlight docs site.
---
# 场景联动

## 一、场景联动介绍
在场景联动中，允许用户依据特定条件和规则，关联并协同控制多个设备的行为。通过预先定义场景联动规则，系统能够自动响应各类事件和状态变化，实现设备间的自动化交互，从而提升系统的智能化与自动化水平。

## 二、功能说明
1. **规则创建与管理**：支持创建、更新和删除场景联动规则，用户可根据实际需求定制自动化逻辑。用户能够灵活定义规则的名称、描述、启用状态和标签等属性，便于对规则进行分类和管理。
2. **多种触发方式**：提供定时任务（`TriggerTypeSchedule`）、设备点位变化（`TriggerTypeDevicePoint`）、设备事件（`TriggerTypeDeviceEvent`）等多种触发方式，以满足不同场景下的触发需求。
3. **条件校验**：系统能够对场景联动规则的执行条件进行严格校验，包括设备点位状态、执行时间、日期区间、年份、月份、日期、星期和时间等多种条件类型，确保规则在合适的条件下执行。
4. **动作执行**：支持执行多种类型的动作，如设置设备点位（`ActionTypeDevicePoint`）、触发其他场景联动（`ActionTypeLinkEdge`）等，实现设备的控制和协同工作。
5. **执行结果记录与反馈**：系统会记录场景联动的执行结果，包括全部成功、部分成功和全部失败三种状态，并通过事件机制进行反馈，方便用户了解执行情况。

## 三、执行流程
1. 系统首先接收触发信号，该信号可能是定时信号、设备点位变化信号或设备事件信号。
2. 系统会校验场景联动规则是否启用，若未启用，则直接记录执行结果并反馈，本次联动结束；若启用，则检查是否处于静默期，若处于静默期，同样记录结果并反馈后结束；
3. 若不在静默期，则校验执行条件，条件满足时执行相应动作，最后记录执行结果并反馈；
4. 若条件不满足，也记录结果并反馈后结束。

## 四、执行条件和动作列举
|类型|详情|说明|
|----|----|----|
|执行条件|设备点位条件（`ConditionTypeDevicePoint`）|检查设备某个点位的值是否满足指定条件，如等于、不等于、大于、小于等。例如，检查温度传感器的温度值是否大于 25℃。|
|执行条件|执行时间条件（`ConditionTypeExecuteTime`）|判断当前时间是否在指定的开始时间和结束时间范围内。例如，设定规则仅在晚上 8 点到 10 点之间执行。|
|执行条件|日期区间条件（`ConditionTypeDateInterval`）|检查当前日期是否在指定的开始日期和结束日期区间内。例如，设置规则在 10 月 1 日到 10 月 7 日之间执行。|
|执行条件|年份条件（`ConditionTypeYears`）|判断当前年份是否符合指定的年份列表。例如，规则仅在 2023 年和 2024 年执行。|
|执行条件|月份条件（`ConditionTypeMonths`）|判断当前月份是否符合指定的月份列表。例如，规则仅在 6 月、7 月、8 月执行。|
|执行条件|日期条件（`ConditionTypeDays`）|判断当前日期是否符合指定的日期列表。例如，规则仅在每个月的 1 号、15 号执行。|
|执行条件|星期条件（`ConditionTypeWeeks`）|判断当前星期是否符合指定的星期列表。例如，规则仅在周一到周五执行。|
|执行条件|时间条件（`ConditionTypeTimes`）|判断当前时间是否在指定的开始时间和结束时间范围内。例如，规则仅在早上 9 点到下午 5 点之间执行。|
|动作|设置设备点位（`ActionTypeDevicePoint`）|对设备的某个点位进行值的设置，可支持单点位和多点位设置。例如，将智能窗帘的开合度设置为 50%。|
|动作|触发其他场景联动（`ActionTypeLinkEdge`）|根据规则触发其他已定义的场景联动规则，实现更复杂的自动化逻辑。例如，一个规则是检测到下雨就关闭窗户，另一个规则是关闭窗户后启动除湿机，当下雨触发第一个规则后，可接着触发第二个规则。|

## 五、关键方法说明

### checkConditions 方法
- **功能**：该方法用于校验场景联动规则的执行条件。它会优先执行点位持续时间条件校验（目前此部分功能未完全实现，暂不影响其他条件校验），然后依次检查各种类型的条件，确保场景联动在满足所有条件时才会执行。
- **参数**：接收一个包含条件的切片 `conditions`，每个条件为 `linkedge.Condition` 类型，包含了条件的类型、具体的条件参数等信息。
- **执行逻辑**：
    - 首先调用 `checkListTimeCondition` 方法进行点位持续时间条件校验，目前该方法直接返回 `nil`，不做实际校验。
    - 遍历 `conditions` 切片，根据不同的条件类型进行相应的校验：
        - 对于设备点位条件（`ConditionTypeDevicePoint`），会检查条件参数的完整性，通过 `helper.DeviceShadow.GetDevicePoint` 获取设备点位的实际值，再调用 `checkConditionValue` 方法比较实际值与条件值是否匹配。
        - 对于执行时间条件（`ConditionTypeExecuteTime`），会获取当前时间的时间戳（`time.Now().UnixMilli()`），与条件中的开始时间和结束时间进行比较，判断当前时间是否在指定范围内。
        - 对于日期区间条件（`ConditionTypeDateInterval`），会先解析开始日期和结束日期，将其转换为 `time.Time` 类型，然后获取当前日期是一年中的第几天（`time.Now().YearDay()`），判断当前日期是否在指定的日期区间内。
        - 对于年份条件（`ConditionTypeYears`），会获取当前年份（`time.Now().Year()`），检查是否在条件指定的年份列表中。
        - 对于月份条件（`ConditionTypeMonths`），会获取当前月份（`int(time.Now().Month())`），检查是否在条件指定的月份列表中。
        - 对于日期条件（`ConditionTypeDays`），会获取当前日期（`time.Now().Day()`），检查是否在条件指定的日期列表中。
        - 对于星期条件（`ConditionTypeWeeks`），会获取当前星期（`int(time.Now().Weekday())`），检查是否在条件指定的星期列表中。
        - 对于时间条件（`ConditionTypeTimes`），会根据条件中的开始时间和结束时间，与当前时间进行比较，判断当前时间是否在指定的时间范围内。
    - 如果所有条件都通过校验，返回 `nil`；只要有一个条件不满足，就返回相应的错误信息。

### TriggerLinkEdge 方法
- **功能**：此方法用于触发指定 ID 的场景联动规则，并记录场景执行记录和更新执行时间。它是整个场景联动执行的入口方法，会调用 `triggerLinkEdge` 方法来实际执行场景联动的逻辑。
- **参数**：接收场景联动的 ID `id` 和触发来源 `source`（目前代码中 `source` 参数的使用不够充分，主要用于记录触发信息）。
- **执行逻辑**：
    - 调用 `triggerLinkEdge` 方法，传入场景联动 ID `id` 和联动深度 `0`，开始执行场景联动的具体逻辑。
    - 如果 `triggerLinkEdge` 方法执行成功，通过 `getLinkEdge` 方法获取场景联动配置，将其执行时间更新为当前时间（`time.Now()`），并将更新后的配置保存到 `configs` 缓存中。
    - 如果 `triggerLinkEdge` 方法执行失败，记录错误日志（`helper.Logger.Error(fmt.Sprintf("linkEdge:%s trigger", e.Error()))`）并返回错误信息。

## 六、API 介绍

### LinkEdgeCreate
- **方法**：`POST`
- **URL 路径**：需从代码中 `route` 包确认 `LinkEdgeCreate` 对应的具体路径，例如可能是 `/api/linkEdge/create`。
- **功能**：创建场景联动规则
- **参数**：`data`（请求体，字节切片），包含场景联动规则的 JSON 数据
- **参数说明**：
    - `name`：场景联动规则的名称，为字符串类型，用于识别和管理规则。
    - `enable`：规则的启用状态，布尔类型，`true` 表示启用，`false` 表示禁用。
    - `description`：规则的描述信息，字符串类型，用于说明规则的用途。
    - `tags`：标签列表，字符串数组，用于对规则进行分类和筛选。
    - `silentPeriod`：静默期，整数类型，单位为秒，在静默期内规则不会重复执行，防止频繁触发。
    - `trigger`：触发器列表，数组类型，每个元素包含 `type`（触发器类型，如 `schedule`、`devicePoint`、`deviceEvent`）及相应的触发器参数（如 `cron` 表达式、设备 ID、点位等）。
    - `condition`：条件列表，数组类型，每个元素包含 `type`（条件类型，如 `devicePoint`、`executeTime` 等）及相应的条件参数。
    - `action`：动作列表，数组类型，每个元素包含 `type`（动作类型，如 `devicePoint`、`linkEdge`）及相应的动作参数（如设备 ID、点位、值、要触发的其他场景联动 ID 等）。
    - **注意**：创建规则时，`action` 列表不能为空，否则会返回 `ErrActionListIsEmpty` 错误。
#### Swagger 示例
```yaml
openapi: 3.0.0
info:
  title: LinkEdgeCreate API
  version: 1.0.0
paths:
  /api/linkEdge/create: # 需根据代码确认实际路径
    post:
      summary: 创建场景联动规则
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                  description: 场景联动规则的名称
                enable:
                  type: boolean
                  description: 规则的启用状态
                description:
                  type: string
                  description: 规则的描述信息
                tags:
                  type: array
                  items:
                    type: string
                  description: 标签列表
                silentPeriod:
                  type: integer
                  description: 静默期，单位为秒
                trigger:
                  type: array
                  items:
                    type: object
                    properties:
                      type:
                        type: string
                        description: 触发器类型
                      cron:
                        type: string
                        description: 定时任务的 cron 表达式，当 type 为 schedule 时必填
                      DeviceID:
                        type: string
                        description: 设备 ID，当 type 为 devicePoint 时必填
                      DevicePoint:
                        type: string
                        description: 设备点位，当 type 为 devicePoint 时必填
                      Condition:
                        type: string
                        description: 条件，如等于、不等于等，当 type 为 devicePoint 时必填
                      Value:
                        type: string
                        description: 条件值，当 type 为 devicePoint 时必填
                    required:
                      - type
                condition:
                  type: array
                  items:
                    type: object
                    properties:
                      type:
                        type: string
                        description: 条件类型
                      DeviceID:
                        type: string
                        description: 设备 ID，当 type 为 devicePoint 时必填
                      DevicePoint:
                        type: string
                        description: 设备点位，当 type 为 devicePoint 时必填
                      Condition:
                        type: string
                        description: 条件，如等于、不等于等，当 type 为 devicePoint 时必填
                      Value:
                        type: string
                        description: 条件值，当 type 为 devicePoint 时必填
                      begin:
                        type: integer
                        description: 开始时间，时间戳，当 type 为 executeTime 时必填
                      end:
                        type: integer
                        description: 结束时间，时间戳，当 type 为 executeTime 时必填
                    required:
                      - type
                action:
                  type: array
                  items:
                    type: object
                    properties:
                      type:
                        type: string
                        description: 动作类型
                      DeviceID:
                        type: string
                        description: 设备 ID，当 type 为 devicePoint 时必填
                      DevicePoint:
                        type: string
                        description: 设备点位，当 type 为 devicePoint 时必填
                      Value:
                        type: string
                        description: 要设置的值，当 type 为 devicePoint 时必填
                      id:
                        type: string
                        description: 要触发的其他场景联动 ID，当 type 为 linkEdge 时必填
                    required:
                      - type
      responses:
        200:
          description: 创建成功
          content:
            application/json:
              schema:
                type: boolean
                example: true
        400:
          description: 创建失败，请求参数错误或其他异常，如 action 列表为空会返回 ErrActionListIsEmpty 错误
          content:
            application/json:
              schema:
                type: object
                properties:
                  error:
                    type: string
```
例如，创建一个规则，在每天晚上 7 点，当客厅温度大于 28℃ 时，打开客厅空调。请求体数据如下：
```json
{
    "name": "晚上高温开空调",
    "enable": true,
    "description": "晚上 7 点，客厅温度大于 28℃ 时打开空调",
    "tags": ["智能家居", "空调控制"],
    "silentPeriod": 300,
    "trigger": [
        {
            "type": "schedule",
            "cron": "0 0 19 * * *"
        },
        {
            "type": "devicePoint",
            "DeviceID": "livingroom_sensor",
            "DevicePoint": "temperature",
            "Condition": ">",
            "Value": "28"
        }
    ],
    "condition": [],
    "action": [
        {
            "type": "devicePoint",
            "DeviceID": "livingroom_airconditioner",
            "DevicePoint": "switch",
            "Value": "on"
        }
    ]
}
```

### LinkEdgeTryTrigger
- **方法**：`POST`
- **URL 路径**：需从代码中 `route` 包确认 `LinkEdgeTryTrigger` 对应的具体路径。
- **功能**：预览联动场景
- **参数**：`data`（请求体，字节切片），包含场景联动规则的 JSON 数据，用于模拟执行
- **参数说明**：与 `LinkEdgeCreate` 的参数说明一致，用于构建模拟执行的场景联动规则。
- **示例**：请求体数据格式同 `LinkEdgeCreate` 的示例，发送 POST 请求到对应的 API 端点，即可预览场景联动执行效果。

### LinkEdgeDelete
- **方法**：`POST`
- **URL 路径**：需从代码中 `route` 包确认 `LinkEdgeDelete` 对应的具体路径。
- **功能**：删除联动场景
- **参数**：`id`（表单参数）
- **参数说明**：要删除的场景联动规则的 ID。
- **示例**：在请求体中设置 `id=scene1`，发送 POST 请求到 API 端点删除 ID 为 `scene1` 的场景联动规则。

### LinkEdgeTrigger
- **方法**：`POST`
- **URL 路径**：需从代码中 `route` 包确认 `LinkEdgeTrigger` 对应的具体路径。
- **功能**：触发联动场景
- **参数**：`id`（表单参数）、`source`（表单参数，可选）
- **参数说明**：`id` 为要触发的场景联动规则的 ID，`source` 为触发来源（目前使用不充分）。
- **示例**：在请求体中设置 `id=scene1&source=manual`，发送 POST 请求到 API 端点触发 ID 为 `scene1` 的场景联动规则。

### LinkEdgeGet
- **方法**：`POST`
- **URL 路径**：需从代码中 `route` 包确认 `LinkEdgeGet` 对应的具体路径。
- **功能**：查看场景联动
- **参数**：`id`（表单参数）
- **参数说明**：要查看的场景联动规则的 ID。