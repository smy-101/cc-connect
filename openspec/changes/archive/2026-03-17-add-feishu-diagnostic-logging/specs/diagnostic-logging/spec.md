## ADDED Requirements

### Requirement: 配置驱动日志级别
系统 SHALL 依据应用配置中的 `log_level` 初始化运行时日志级别，并使该级别影响 CLI 启动后的结构化日志输出。

#### Scenario: 使用 debug 级别启动
- **WHEN** 应用配置中的 `log_level` 被设置为 `debug`
- **THEN** 系统输出包含 debug 级别的诊断日志

#### Scenario: 使用 info 级别启动
- **WHEN** 应用配置中的 `log_level` 被设置为 `info`
- **THEN** 系统不输出 debug 级别的诊断日志

### Requirement: 飞书入站链路可诊断
系统 SHALL 为飞书入站消息链路输出结构化诊断日志，以便操作者判断事件是否到达、是否完成转换以及是否进入路由处理。

#### Scenario: 收到真实文本事件
- **WHEN** 飞书长连接已就绪且系统收到真实 `im.message.receive_v1` 文本事件
- **THEN** 日志中记录事件到达阶段及 `event_id`、`message_id`、`chat_type`、`message_type` 等关键字段

#### Scenario: 事件转换失败
- **WHEN** 收到的飞书事件无法转换为统一消息
- **THEN** 日志中记录转换失败阶段与错误原因

#### Scenario: 路由处理失败
- **WHEN** 飞书事件完成转换但路由处理返回错误
- **THEN** 日志中记录路由失败阶段与错误原因

### Requirement: 回复与代理调用可诊断
系统 SHALL 为状态回复、Claude Code 调用和最终回复发送输出阶段化诊断日志，以便区分飞书入站问题与代理调用问题。

#### Scenario: 发送状态回复
- **WHEN** 文本消息进入应用处理器并准备发送“正在思考”状态回复
- **THEN** 日志中记录状态回复发送阶段与目标会话标识

#### Scenario: Claude Code 调用失败
- **WHEN** Claude Code 调用失败或超时
- **THEN** 日志中记录代理调用阶段与失败结果

#### Scenario: 最终回复发送失败
- **WHEN** 系统已获得代理结果但向飞书发送最终回复失败
- **THEN** 日志中记录最终回复发送阶段、目标会话标识与错误原因

### Requirement: 敏感信息最小暴露
系统 SHALL 在诊断日志中避免输出敏感配置与完整消息正文，仅输出联调所需的标识性字段与错误信息。

#### Scenario: 记录飞书链路日志
- **WHEN** 系统记录飞书连接、事件接收或发送相关日志
- **THEN** 日志不包含 `app_secret` 或其它敏感凭证明文

#### Scenario: 记录消息处理日志
- **WHEN** 系统记录用户消息或回复处理日志
- **THEN** 日志默认不包含完整消息正文，而仅包含必要元数据或长度信息
