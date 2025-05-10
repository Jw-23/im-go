<!-- @formatter:off -->
# API 文档

本文档定义了 IM 项目的 HTTP RESTful API，方便前后端对接。

## HTTP RESTful API 文档

**基础 URL**：假设 API 服务器运行在 `http://localhost:8081` (根据 `config.API_SERVER.PORT`)

**认证**：
*   需要认证的 API 端点，应在请求头中包含 `Authorization: Bearer <YOUR_JWT_TOKEN>`。
*   JWT Token 通过 `/auth/login` 端点获取。

**通用响应格式**：
*   **成功**:
    ```json
    {
        // "data": { ... } // 具体的响应数据
        // 或直接是数据对象/列表
    }
    ```
    HTTP 状态码: `200 OK`, `201 Created`, `204 No Content` 等。
*   **失败**:
    ```json
    {
        "error": "错误描述信息"
    }
    ```
    HTTP 状态码: `400 Bad Request`, `401 Unauthorized`, `403 Forbidden`, `404 Not Found`, `500 Internal Server Error` 等。

---

### 1. 认证 (Auth)

#### 1.1 用户注册

*   **Endpoint**: `POST /auth/register`
*   **描述**: 注册新用户。
*   **认证**: 公开
*   **请求体** (`application/json`):
    ```json
    {
        "username": "string (required, unique)",
        "password": "string (required, min length typically 6-8)",
        "email": "string (required, unique, valid email format)",
        "nickname": "string (optional)"
    }
    ```
*   **成功响应** (`201 Created`):
    ```json
    {
        "id": "uint (用户ID)",
        "username": "string",
        "email": "string",
        "nickname": "string",
        "avatarUrl": "string",
        "createdAt": "time.Time"
    }
    ```
    (返回新创建用户的信息，不含密码)
*   **错误响应**:
    *   `400 Bad Request`: 请求体无效，或必填字段缺失/格式错误 (例如，用户名已存在，邮箱已存在)。
    *   `500 Internal Server Error`: 服务器内部错误。

#### 1.2 用户登录

*   **Endpoint**: `POST /auth/login`
*   **描述**: 用户登录，获取 JWT。
*   **认证**: 公开
*   **请求体** (`application/json`):
    ```json
    {
        "username": "string (required)",
        "password": "string (required)"
    }
    ```
*   **成功响应** (`200 OK`):
    ```json
    {
        "token": "string (JWT)"
    }
    ```
*   **错误响应**:
    *   `400 Bad Request`: 请求体无效。
    *   `401 Unauthorized`: 用户名或密码错误。
    *   `500 Internal Server Error`: 服务器内部错误。

---

### 2. 用户 (Users)

#### 2.1 获取当前用户信息

*   **Endpoint**: `GET /api/v1/users/me`
*   **描述**: 获取当前已认证用户的个人资料。
*   **认证**: JWT 必需
*   **成功响应** (`200 OK`):
    ```json
    // models.User 结构 (不含 PasswordHash)
    {
        "id": "uint",
        "username": "string",
        "email": "string",
        "nickname": "string",
        "avatarUrl": "string",
        "status": "string",
        "lastSeenAt": "time.Time | null",
        "bio": "string",
        "createdAt": "time.Time",
        "updatedAt": "time.Time"
    }
    ```
*   **错误响应**:
    *   `401 Unauthorized`: 未认证或 Token 无效。
    *   `404 Not Found`: 用户未找到 (理论上对于 `/me` 不会发生，除非DB数据不一致)。

#### 2.2 更新当前用户信息

*   **Endpoint**: `PUT /api/v1/users/me`
*   **描述**: 更新当前已认证用户的个人资料。
*   **认证**: JWT 必需
*   **请求体** (`application/json`):
    ```json
    {
        "nickname": "string (optional)",
        "avatarUrl": "string (optional)",
        "bio": "string (optional)"
        // 注意：username, email, password 通常通过其他专用端点修改
    }
    ```
*   **成功响应** (`200 OK`):
    ```json
    // 更新后的 models.User 结构 (不含 PasswordHash)
    {
        "id": "uint",
        "username": "string",
        "email": "string",
        "nickname": "string",
        "avatarUrl": "string",
        "status": "string",
        "lastSeenAt": "time.Time | null",
        "bio": "string",
        "createdAt": "time.Time",
        "updatedAt": "time.Time"
    }
    ```
*   **错误响应**:
    *   `400 Bad Request`: 请求体无效。
    *   `401 Unauthorized`: 未认证或 Token 无效。
    *   `500 Internal Server Error`: 更新失败。

#### 2.3 获取指定用户信息

*   **Endpoint**: `GET /users/{userID}`
*   **描述**: 获取指定用户的公开信息。
*   **认证**: 公开 (或可选 JWT，用于未来可能的权限控制)
*   **URL 参数**:
    *   `userID`: `uint` - 要获取信息的用户的 ID。
*   **成功响应** (`200 OK`):
    ```json
    // models.User 公开字段 (例如，不含 email，除非特别设计)
    {
        "id": "uint",
        "username": "string",
        "nickname": "string",
        "avatarUrl": "string",
        "status": "string", // 可能只显示 online/offline
        "bio": "string"
    }
    ```
*   **错误响应**:
    *   `404 Not Found`: 用户未找到。

---

### 3. 会话 (Conversations)

#### 3.1 获取当前用户的会话列表

*   **Endpoint**: `GET /api/v1/conversations`
*   **描述**: 获取当前认证用户参与的所有会话列表。
*   **认证**: JWT 必需
*   **Query 参数**:
    *   `limit`: `int` (optional, default: e.g., 20) - 每页数量。
    *   `offset`: `int` (optional, default: 0) - 偏移量。
*   **成功响应** (`200 OK`):
    ```json
    [
        // 列表，每个元素是会话的详细信息
        {
            "id": "uint (会话ID)",
            "type": "string ('private' or 'group')",
            "targetId": "uint (私聊时为对方用户ID, 群聊时为GroupID)",
            "name": "string (私聊时为对方昵称, 群聊时为群名称)",
            "avatar": "string (私聊时为对方头像URL, 群聊时为群头像URL)",
            "lastMessage": {
                "id": "uint (消息ID)",
                "content": "string (消息内容)",
                "timestamp": "time.Time (消息发送时间)",
                "senderId": "string (发送者ID)"
            } | null, // 如果没有最后消息则为null
            "updatedAt": "time.Time (会话最后更新时间，用于排序)",
            "unreadCount": "int (未读消息数, 预留字段，当前可能未实现或为0)"
        }
    ]
    ```
*   **错误响应**:
    *   `401 Unauthorized`: 未认证。

#### 3.2 创建或获取私聊会话

*   **Endpoint**: `POST /api/v1/conversations/private`
*   **描述**: 与指定用户开始私聊。如果已存在会话，则返回现有会话。
*   **认证**: JWT 必需
*   **请求体** (`application/json`):
    ```json
    {
        "targetId": "uint (required, 对方用户的 ID)"
    }
    ```
*   **成功响应** (`200 OK` 或 `201 Created`):
    ```json
    // models.Conversation 结构
    {
        "id": "uint",
        "type": "private",
        // ... 其他会话字段
    }
    ```
*   **错误响应**:
    *   `400 Bad Request`: 请求体无效或目标用户ID缺失。
    *   `401 Unauthorized`: 未认证。
    *   `404 Not Found`: 目标用户未找到。
    *   `500 Internal Server Error`: 创建失败。

#### 3.3 获取会话消息

*   **Endpoint**: `GET /api/v1/conversations/{conversationID}/messages`
*   **描述**: 获取指定会话的消息列表（分页）。
*   **认证**: JWT 必需 (需要验证用户是否是会话参与者)
*   **URL 参数**:
    *   `conversationID`: `uint` - 会话 ID。
*   **Query 参数**:
    *   `limit`: `int` (optional, default: e.g., 50) - 每页数量。
    *   `offset`: `int` (optional, default: 0) - 偏移量 (或者使用 `beforeMessageID` 进行游标分页)。
*   **成功响应** (`200 OK`):
    ```json
    [
        // 列表，每个元素是 models.Message 结构，按时间倒序排列
        {
            "id": "uint",
            "conversationId": "uint",
            "senderId": "uint",
            "type": "string (MessageTypeDB)",
            "content": "string (文本内容或文件URL)",
            "metadata": {
                // "fileName": "string", (if file/image)
                // "fileSize": "int64" (if file/image)
            },
            "sentAt": "time.Time",
            "status": "string ('sent', 'delivered', 'read')"
            // "sender": { "id": "uint", "nickname": "string", "avatarUrl": "string" } // (可选) 预加载发送者信息
        }
    ]
    ```
*   **错误响应**:
    *   `401 Unauthorized`: 未认证。
    *   `403 Forbidden`: 用户无权访问此会话的消息。
    *   `404 Not Found`: 会话未找到。

---

### 4. 群组 (Groups)

#### 4.1 创建群组

*   **Endpoint**: `POST /api/v1/groups`
*   **描述**: 创建一个新的群组。
*   **认证**: JWT 必需
*   **请求体** (`application/json`): (参考 `apiserver.CreateGroupRequest`)
    ```json
    {
        "name": "string (required, 群组名称)",
        "description": "string (optional, 群组描述)",
        "avatarUrl": "string (optional, 群组头像URL)",
        "isPublic": "bool (default: false, 是否为公开群组)",
        "joinCondition": "string (optional, e.g., 'direct', 'approval_required')"
    }
    ```
*   **成功响应** (`201 Created`):
    ```json
    // models.Group 结构
    {
        "id": "uint",
        "name": "string",
        "description": "string",
        "avatarUrl": "string",
        "ownerId": "uint (创建者ID)",
        "memberCount": "int (初始为1)",
        "isPublic": "bool",
        "joinCondition": "string",
        "createdAt": "time.Time"
    }
    ```
*   **错误响应**:
    *   `400 Bad Request`: 请求体无效。
    *   `401 Unauthorized`: 未认证。
    *   `500 Internal Server Error`: 创建失败。

#### 4.2 获取群组详情

*   **Endpoint**: `GET /groups/{groupID}`
*   **描述**: 获取指定群组的详细信息。
*   **认证**: 公开 (如果群组是公开的) 或 JWT 必需 (如果群组是私有的，需要验证用户是否为成员)
*   **URL 参数**:
    *   `groupID`: `uint` - 群组 ID。
*   **成功响应** (`200 OK`):
    ```json
    // models.Group 结构
    {
        "id": "uint",
        "name": "string",
        "description": "string",
        "avatarUrl": "string",
        "ownerId": "uint",
        "memberCount": "int",
        "isPublic": "bool",
        "joinCondition": "string",
        "createdAt": "time.Time"
        // "members": [ ... ] // (可选) 部分成员列表预览
    }
    ```
*   **错误响应**:
    *   `403 Forbidden`: (如果私有群组且用户非成员)
    *   `404 Not Found`: 群组未找到。

#### 4.3 加入群组

*   **Endpoint**: `POST /api/v1/groups/{groupID}/join`
*   **描述**: 当前用户加入指定群组。
*   **认证**: JWT 必需
*   **URL 参数**:
    *   `groupID`: `uint` - 要加入的群组 ID。
*   **成功响应** (`200 OK`):
    ```json
    // models.GroupMember 结构 (表示用户已成为成员)
    {
        "groupId": "uint",
        "userId": "uint",
        "role": "string (e.g., 'member')",
        "joinedAt": "time.Time"
    }
    // 或仅返回成功消息
    // { "message": "成功加入群组" }
    ```
*   **错误响应**:
    *   `401 Unauthorized`: 未认证。
    *   `403 Forbidden`: 群组不允许加入 (例如私有、需要审批且未通过)。
    *   `404 Not Found`: 群组未找到。
    *   `409 Conflict`: 用户已是群组成员。
    *   `500 Internal Server Error`: 操作失败。

#### 4.4 离开群组

*   **Endpoint**: `POST /api/v1/groups/{groupID}/leave`
*   **描述**: 当前用户离开指定群组。
*   **认证**: JWT 必需
*   **URL 参数**:
    *   `groupID`: `uint` - 要离开的群组 ID。
*   **成功响应** (`200 OK`):
    ```json
    {
        "message": "成功离开群组"
    }
    ```
*   **错误响应**:
    *   `401 Unauthorized`: 未认证。
    *   `403 Forbidden`: 用户不是群组成员 (或群主试图离开未解散/转移所有权的群)。
    *   `404 Not Found`: 群组未找到。
    *   `500 Internal Server Error`: 操作失败。

#### 4.5 获取群组成员列表

*   **Endpoint**: `GET /api/v1/groups/{groupID}/members`
*   **描述**: 获取指定群组的成员列表。
*   **认证**: JWT 必需 (需要验证用户是否为群组成员，或群组是否公开允许查看成员)
*   **URL 参数**:
    *   `groupID`: `uint` - 群组 ID。
*   **Query 参数**:
    *   `limit`: `int` (optional, default: e.g., 50)
    *   `offset`: `int` (optional, default: 0)
*   **成功响应** (`200 OK`):
    ```json
    [
        // 列表，每个元素是 models.GroupMember 结构，可预加载 User 信息
        {
            "userId": "uint",
            "role": "string",
            "joinedAt": "time.Time",
            "alias": "string (群内昵称)",
            "user": {
                "id": "uint",
                "nickname": "string",
                "avatarUrl": "string"
            }
        }
    ]
    ```
*   **错误响应**:
    *   `401 Unauthorized`: 未认证。
    *   `403 Forbidden`: 无权查看成员列表。
    *   `404 Not Found`: 群组未找到。

#### 4.6 搜索公开群组

*   **Endpoint**: `GET /groups/search`
*   **描述**: 搜索公开的群组。
*   **认证**: 公开
*   **Query 参数**:
    *   `q`: `string` (required, 搜索关键词，匹配群名或描述)
    *   `limit`: `int` (optional, default: e.g., 20)
    *   `offset`: `int` (optional, default: 0)
*   **成功响应** (`200 OK`):
    ```json
    [
        // models.Group 结构列表
        {
            "id": "uint",
            "name": "string",
            "description": "string",
            "avatarUrl": "string",
            "memberCount": "int",
            "isPublic": true
        }
    ]
    ```

---
<!-- @formatter:on --> 