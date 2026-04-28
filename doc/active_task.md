# Active Task: 渠道管理页面左侧操作 Rail

## Context
重构渠道管理页面操作入口：把所有行级操作和全局操作收拢到页面最左侧的固定垂直工具栏（Rail）。Rail 永远固定在视口左侧、不随表格滚动条移动；图标作用于"当前选中行/批量选中行"，多选时单行专属图标置灰。删除表格右侧 OPERATE 列。模型 CRUD 通过展开渠道行查看模型子表 + 选中模型行后用 Rail 操作。

## Critical Files
- web/src/components/table/channels/index.jsx
- web/src/components/table/channels/ChannelsActionRail.jsx (新建)
- web/src/components/table/channels/ChannelsColumnDefs.jsx
- web/src/components/table/channels/ChannelsTable.jsx
- web/src/components/table/channels/ChannelsActions.jsx
- web/src/hooks/channels/useChannelsData.jsx

## Step-by-Step Plan

### 阶段 1：Rail 立起来 + 删除 OPERATE 列 ✅
- [x] 1. 新建 `ChannelsActionRail.jsx`：56px 宽垂直工具栏，sticky 在页面左侧。图标分四组（细分隔线分隔）：
  - **全局**：新增渠道、新增模型、刷新
  - **行级**：测试、启用/禁用、编辑、复制、删除
  - **条件**：多密钥管理（仅 multi-key 渠道行）、Ollama 测活
  - **工具**：批量测活、紧凑列表、列设置
- [x] 2. 修改 `index.jsx`：用 flex 布局把 Rail 放在 CardPro 左边
- [x] 3. 修改 `ChannelsColumnDefs.jsx`：删除 OPERATE 整列定义 + COLUMN_KEYS 中的 OPERATE 项 + 不再使用的 imports（Dropdown、SplitButtonGroup、IconMore）
- [x] 4. 修改 `useChannelsData.jsx`：把 `enableBatchDelete` 强制为 `true`，让 checkbox 永远可见
- [x] 5. 修改 `ChannelsActions.jsx`：删除已迁移到 Rail 的按钮（删除所选/批量操作下拉里的"测试所有"/紧凑列表/批量操作开关），保留维护操作下拉 + 设置开关
- [x] 6. Rail 图标 disabled 逻辑实现：
  - 0 选中：只启用全局组 + 工具组
  - 1 选中（非 tag 行）：启用所有相关图标（多密钥/Ollama 按行类型再次过滤）
  - 2+ 选中：禁用单选专属图标（编辑、复制、多密钥、Ollama），保留批量兼容（测试、启停、删除）

### 阶段 2：模型行展开 + 模型 CRUD ✅
- [x] 7. `ChannelsTable.jsx`：启用 Semi Table 的 `expandedRowRender` + `rowExpandable`，受控 `expandedRowKeys`，渲染 `ChannelModelsSubTable`
- [x] 8. 新建 `ChannelModelsSubTable.jsx`：解析 `channel.models` 逗号串 + `JSON.parse(channel.model_mapping)`；列：模型名 / 重定向到。子表自带 rowSelection
- [x] 9. Rail 改造：双轨选中（selectedChannels + selectedModels）；mode = 'channel' / 'model' / 'mixed' / 'none' 决定 dispatch
- [x] 10. 新增 `selectedModels` 状态、`setChannelModelSelection`（每子表写自己片段）、`removeModelsFromChannel`（PUT /api/channel/ 改 models + model_mapping）于 useChannelsData
- [x] 11. Rail handler 支持的语义：
  - **新增模型**：选中 1 渠道 或 1 模型 → 打开父渠道 EditChannelModal
  - **测试**：渠道行 → testChannel(ch, '')；模型行 → testChannel(ch, modelName)
  - **编辑**：渠道行 → 打开渠道编辑器；模型行 → 打开父渠道编辑器（用户在 models 字段调）
  - **删除**：渠道行 → batchDeleteChannels；模型行 → 按渠道分组调用 removeModelsFromChannel
  - **启停/复制/多密钥/Ollama**：仅渠道行，模型选中时置灰

### 阶段 3：验证
- [x] 11. `bun install` + `bun run build` 通过，无编译错误（控制台 warning 全是 React 18 deprecated API，与本任务无关）
- [x] 12. `bun run dev` 启动 Vite，无 HMR/转译错误。Playwright 访问 `/` 能加载 React 应用
- [ ] 13. 浏览器深度验证（依赖后端可用）：未做，本机后端未启动，进 `/console/channel` 会被 Zitadel auth 拦截。需用户在完整环境（前后端齐启）里跑

## Current Status
- [x] 阶段 1 + 阶段 2 代码全部改完
- [x] 编译/转译验证通过
- [ ] 待用户在完整环境跑通后逐项验证（见下方"用户验证 checklist"）

## 用户验证 checklist
1. Rail 在视口左侧，56px 宽，**滚表格时不动**（横/竖都不动）
2. 0 选中：测试/启停/编辑/复制/删除/多密钥/Ollama 全灰；新增渠道/新增模型(灰)/刷新/批量测试/紧凑/列设置 亮
3. 勾选 1 个 OpenAI 渠道：编辑/复制/删除/测试/启停 亮；多密钥/Ollama 灰
4. 勾选 1 个 Ollama 渠道：Ollama 测活图标亮起
5. 勾选 1 个多密钥渠道：多密钥图标亮起
6. 勾选 2+ 渠道：编辑/复制/多密钥/Ollama 灰；测试/启停/删除 亮
7. 渠道行点 ▶ 展开 → 模型子表显示模型名 + 重定向映射
8. 子表勾选 1 个模型：Rail 测试/编辑/删除 亮；启停/复制/多密钥/Ollama 灰
9. 同时勾选渠道 + 模型 → 单行图标全部灰（"混合选中渠道+模型时不可用"）
10. 模型行点删除 → PUT /api/channel/ 把 model 从 channel.models 拿掉，刷新后看不到该模型
11. 模型行点编辑 → 弹出该模型所属渠道的编辑器
12. 表格右侧 OPERATE 列彻底消失，无残留按钮
