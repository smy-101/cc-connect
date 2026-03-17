# slash-commands 规格 (Delta)

## Purpose

斜杠命令系统提供用户与 cc-connect 交互的主要方式。本次变更新增 `/project` 命令支持多项目切换。

## ADDED Requirements

### Requirement: /project 命令

系统 SHALL 支持 `/project` 命令管理多项目。

#### Scenario: 显示当前项目
- **WHEN** 用户发送 `/project`（无参数）
- **THEN** 系统 SHALL 返回当前项目信息
- **AND** 系统 SHALL 返回可用项目列表
- **AND** 当前项目 SHALL 有明确标识

#### Scenario: 切换项目
- **WHEN** 用户发送 `/project <name>`
- **THEN** 系统 SHALL 切换到指定项目
- **AND** 系统 SHALL 清除旧会话（默认行为）
- **AND** 系统 SHALL 返回切换结果

#### Scenario: 切换项目并保留会话
- **WHEN** 用户发送 `/project <name> --keep`
- **THEN** 系统 SHALL 切换到指定项目
- **AND** 系统 SHALL 保留旧项目的会话

#### Scenario: 短标志保留会话
- **WHEN** 用户发送 `/project <name> -k`
- **THEN** 系统 SHALL 切换到指定项目
- **AND** 系统 SHALL 保留旧项目的会话

### Requirement: 命令标志解析

系统 SHALL 支持命令标志解析。

#### Scenario: 解析长标志
- **WHEN** 解析 `/project backend --keep`
- **THEN** 系统 SHALL 返回 Command{Name: "project", Args: ["backend"], Flags: {"keep": "true"}}

#### Scenario: 解析短标志
- **WHEN** 解析 `/project backend -k`
- **THEN** 系统 SHALL 返回 Command{Name: "project", Args: ["backend"], Flags: {"k": "true"}}

#### Scenario: 无标志命令
- **WHEN** 解析 `/project backend`
- **THEN** 系统 SHALL 返回 Command{Name: "project", Args: ["backend"], Flags: {}}

## MODIFIED Requirements

### Requirement: /help 命令

系统 SHALL 支持 `/help` 命令显示可用命令的帮助信息。

#### Scenario: 显示帮助
- **WHEN** 用户发送 `/help`
- **THEN** 系统 SHALL 返回所有可用命令的列表
- **AND** 列表 SHALL 包含 /mode、/new、/list、/help、/stop、**/project**
- **AND** 每个命令 SHALL 有简要说明

#### Scenario: /project 帮助信息
- **WHEN** 用户发送 `/help`
- **THEN** 帮助信息 SHALL 包含 /project 命令说明
- **AND** 说明 SHALL 包含用法示例
