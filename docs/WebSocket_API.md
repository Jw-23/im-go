<!-- @formatter:off -->
## WebSocket API 文档

**连接 URL**: `ws://CHAT_SERVER_HOST:CHAT_SERVER_PORT/ws/chat?token=<YOUR_JWT_TOKEN>`
*   `CHAT_SERVER_HOST:CHAT_SERVER_PORT` 是聊天服务器的地址和端口 (根据 `config.SERVER.HOST` 和 `config.SERVER.PORT`，例如 `localhost:8080`)。
*   路径是 `/ws/chat` (根据 `config.SERVER.WEBSOCKET_PATH`)。
*   `token=<YOUR_JWT_TOKEN>`: JWT Token 作为查询参数进行认证。如果 Token 无效或缺失，连接可能会被拒绝，或用户被视为匿名（取决于服务器配置）。

### 消息格式

所有通过 WebSocket 交换的消息都应为 JSON 格式，并遵循 `imtypes.Message` 结构。

#### `imtypes.Message` 结构:

```typescript
interface WebSocketMessage {
    id?: string;             // 客户端生成的消息ID (可选, UUID)，服务端会生成自己的持久化ID
    type: MessageType;       // 消息类型 (见下文)
    content: string;         // 消息内容 (文本、JSON序列化的媒体信息、或由type决定的其他内容)
                             // 对于图片/文件，这里通常是上传后返回的 URL 或相关元数据 JSON 字符串
    senderId?: string;       // 发送者ID (客户端发送时可不填或填自己的，服务端会根据token覆盖)
    receiverId: string;      // 接收者ID (私聊时为对方 UserID，群聊时为 GroupID/ConversationID)
    timestamp?: string;      // 客户端发送时间 (ISO 8601 string, 可选，服务端会记录接收时间)
    fileName?: string;       // (可选) 文件名，当 type 为 "file" 或 "image"
    fileSize?: number;       // (可选) 文件大小 (字节)，当 type 为 "file" 或 "image"
    conversationId?: string; // (可选) 消息所属的会话ID。客户端发送私聊消息时，如果不知道 conversationId，可以只填 receiverId。
                             // 服务端下发消息时，此字段通常会包含。
}

enum MessageType {
    TEXT = "text",
    IMAGE = "image",         // 内容可以是图片URL或元数据
    FILE = "file",           // 内容可以是文件URL或元数据
    EMOJI = "emoji",
    SYSTEM = "system"        // 系统消息 (例如，用户加入/离开群聊，由服务器发送)
    // 后续可扩展: audio, video, typing_indicator, read_receipt
}
```
**注意**: `content` 字段的实际内容取决于 `type`。对于媒体消息（图片、文件），建议的流程是：
1.  客户端通过 HTTP API (未在此文档中详述，通常是 `/api/v1/upload/file` 或类似接口) 上传文件。
2.  上传接口返回文件的 URL 和其他元数据 (如文件名、大小)。
3.  客户端构建 `WebSocketMessage`，将 `type` 设置为 `image` 或 `file`，并将 URL 或元数据 JSON 字符串放入 `content` 字段，同时填充 `fileName` 和 `fileSize`。

### 1. 客户端发送给服务器的消息

当客户端通过 WebSocket 连接向服务器发送消息时：
*   **必填字段**:
    *   `type`: 消息类型。
    *   `content`: 消息内容。
    *   `receiverId`:
        *   **私聊**: 目标用户的 UserID (字符串形式)。
        *   **群聊**: 群组的 ConversationID (通常与 GroupID 对应，需确认具体实现，字符串形式)。
*   **可选字段**:
    *   `id`: 客户端生成的消息唯一标识，用于去重或追踪。
    *   `timestamp`: 客户端消息发送时间。
    *   `fileName`, `fileSize`: 文件类型消息的元数据。
    *   `conversationId`: 如果是回复特定会话，可以带上。
*   **服务端处理**:
    *   服务器会验证 `SenderID` (根据连接的JWT Token)。
    *   服务器会记录准确的接收时间。
    *   消息会被存储，并通过 Kafka 进一步处理和分发。

**示例 (客户端发送文本消息给用户 "123")**:
```json
{
    "type": "text",
    "content": "你好啊！",
    "receiverId": "123"
}
```

**示例 (客户端发送图片消息到会话/群 "group_abc" 或 "conversation_xyz")**:
```json
{
    "type": "image",
    "content": "{\"url\": \"https://example.com/path/to/image.jpg\", \"thumbnailUrl\": \"...\"}", // 或仅 URL 字符串
    "receiverId": "conversation_xyz", // 或群聊的 ConversationID
    "fileName": "cute_cat.jpg",
    "fileSize": 102400
}
```

### 2. 服务器发送给客户端的消息

当服务器通过 WebSocket 向客户端推送消息时 (包括自己发送的消息的回执、他人发送的消息、系统通知等)：
*   消息结构遵循 `imtypes.Message`。
*   **所有字段通常都由服务端填充**:
    *   `id`: 数据库中该消息的唯一 ID (通常是 `uint` 转为 `string`)。
    *   `type`: 消息类型。
    *   `content`: 消息内容。
    *   `senderId`: 发送者的 UserID (字符串形式)。
    *   `receiverId`:
        *   对于私聊消息，这里是接收此推送的客户端的 UserID。
        *   对于群聊消息，这里也是接收此推送的客户端的 UserID (服务器会为群内每个在线成员单独推送)。
    *   `timestamp`: 消息在服务端的发送/入库时间 (ISO 8601 格式)。
    *   `fileName`, `fileSize`: (如果适用)。
    *   `conversationId`: 此消息所属的会话 ID (字符串形式)。

**示例 (服务器推送一条来自用户 "456" 的文本消息给当前客户端)**:
```json
{
    "id": "789", // 服务端数据库消息ID
    "type": "text",
    "content": "Hello from user 456!",
    "senderId": "456",
    "receiverId": "当前客户端的UserID", // 由服务器在分发时确定
    "timestamp": "2023-10-27T10:30:00Z",
    "conversationId": "private_conv_id_between_current_and_456"
}
```

**示例 (服务器推送一条群消息)**:
```json
{
    "id": "800",
    "type": "text",
    "content": "大家好，我是群里的新人！",
    "senderId": "777", // 发送者 UserID
    "receiverId": "当前客户端的UserID", //  这条消息是推送给"当前客户端的UserID"的
    "timestamp": "2023-10-27T11:00:00Z",
    "conversationId": "group_conv_id_101" // 群的 ConversationID
}
```

---
<!-- @formatter:on --> 