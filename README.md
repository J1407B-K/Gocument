# Gocument 在线多人文档协作平台

###### 简要概述：

一个非常简单的简易在线多人文档协作平台(以前端实现文本编辑器为前提，向后端传递doc文件)

初始化配置（config.yaml）在manifest文件夹下

第一次跑请go run ./app/api/main.go --db初始化数据库 



**Gocument 是一个基于 Golang 与 Gin 框架构建的文档管理服务，提供用户注册、登录、头像上传、文档上传/更新/删除、以及文档与头像的访问接口。服务中集成了 MySQL、Redis 与腾讯云 COS 用于数据持久化、缓存与文件存储。**



已实现功能：

- [x] 用户注册与登录（包括加盐加密、Jwt）
- [x] 文档创建/单人编辑（暂时只实现腾讯COS上传和拉取文档/头像的接口）
- [x] 文档查看/分享（基于COS连接）
- [x]  文档列表
- [x] Redis缓存
- [x] 简易权限（通过mysql的file_access表完成，并未深入，同时通过gorm的钩子函数与文件元数据同步刷新）
- [x] 配置文件（Viper）
- [x] 日志管理（Zap）
- [x] 简单WebSocket连接（真的很简单！就建立了个连接hhhh）

---

## 目录

- [功能简介](#功能简介)
- [技术栈](#技术栈)
- [接口示范](#接口示范)
- [备注](#备注)

---

## 功能简介

- **用户注册/登录**  
  用户可通过 JSON 格式提交用户名和密码进行注册与登录。密码会使用 bcrypt 加密存储，同时通过 Redis 与 MySQL 双重检测用户是否已存在。

- **头像管理**  
  用户可上传头像（仅支持 `.jpg` 格式），头像文件将上传至腾讯云 COS 并保存元数据至 MySQL，同时在 Redis 中做缓存。

- **文档管理**  
  支持文档上传（仅支持 `.docx` 格式），上传时可设置文档的可见性（默认为 `public`），同时支持文档的获取、删除及更新操作。更新操作中会先删除旧的文档，再上传新的文件，并更新数据库和缓存。

- **文件权限与鉴权**  
  通过中间件将已登录用户名注入 `context`，接口内部根据当前用户与文件元数据进行鉴权。

---

## 技术栈
- **语言**：Golang
- **Web 框架**：Gin
- **数据库**：MySQL（gorm操作，存储用户信息与文件元数据）
- **缓存**：Redis（存储用户信息及文件路径缓存）
- **对象存储**：腾讯云 COS
- **加密**：bcrypt（密码加密）
- **日志**：Zap
- **配置管理**：Viper

## 接口示范（CHATGPT生成，可能有错）

### 1. 用户注册

- **接口地址**: `POST /register`

- **功能说明**:
  用户注册接口，接收 JSON 格式的用户名和密码。接口会先在 Redis 与 MySQL 中检查用户是否已存在，密码会使用 bcrypt 加密后存储。

- 请求示例:

  - Headers:

    - `Content-Type: application/json`

  - Body:

    ```
    json复制编辑{
      "username": "alice",
      "password": "123456"
    }
    ```

- 返回示例:

  - 成功:

    ```
    json复制编辑{
      "code": 0,
      "msg": "user register success"
    }
    ```

  - 错误 (如用户已存在):

    ```
    json复制编辑{
      "code": 1001,
      "msg": "user already exist"
    }
    ```

------

### 2. 用户登录

- **接口地址**: `POST /login`

- **功能说明**:
  用户登录接口，验证用户名和密码。如果登录成功，将生成 JWT 令牌返回给客户端。

- 请求示例:

  - Headers:

    - `Content-Type: application/json`

  - Body:

    ```
    json复制编辑{
      "username": "alice",
      "password": "123456"
    }
    ```

- 返回示例:

  - 成功:

    ```
    json复制编辑{
      "code": 0,
      "msg": "login success",
      "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6..."
    }
    ```

  - 错误 (如密码不正确):

    ```
    json复制编辑{
      "code": 1002,
      "msg": "password compare failed: <错误详细信息>"
    }
    ```

------

### 3. 获取用户信息

- **接口地址**: `GET /select`

- **功能说明**:
  根据查询参数 `username` 返回用户基本信息及其上传的文件元数据（包括文档和头像）。

- 请求示例:

  - Query 参数:
    - `username`（必填）：用户名，例如 `/select?username=alice`

- 返回示例:

  ```
  json复制编辑{
    "code": 0,
    "msg": "Info selecting successfully",
    "User": {
      "username": "alice",
      "password": "-"  // 密码经过屏蔽处理
      // 其他用户信息字段...
    },
    "MetaFiles": [
      {
        "FileName": "document1.docx",
        "FileURL": "documents/alice/document1.docx",
        "Visibility": "public",
        "Username": "alice"
      },
      {
        "FileName": "alice.jpg",
        "FileURL": "avatars/alice/alice.jpg",
        "Visibility": "public",
        "Username": "alice"
      }
    ]
  }
  ```

------

### 4. 获取头像

- **接口地址**: `GET /get/avatar`

- **功能说明**:
  根据查询参数 `username` 获取用户头像文件的 URL。头像文件名默认为 `<username>.jpg`。

- 请求示例:

  - Query 参数:
    - `username`（必填）：例如 `/get/avatar?username=alice`

- 返回示例:

  ```
  json复制编辑{
    "code": 0,
    "msg": "avatar get successfully",
    "URL": "https://<your-cos-domain>/avatars/alice/alice.jpg"
  }
  ```

------

## 受保护接口 (需要 JWT 认证)

以下接口需要在请求头中携带有效的 JWT 令牌：

```
makefile


复制编辑
Authorization: Bearer <jwt_token>
```

### 5. 上传头像

- **接口地址**: `POST /upload/avatar`

- **功能说明**:
  上传用户头像接口，只允许上传 `.jpg` 格式的头像。上传的头像文件会存储到腾讯云 COS，同时保存文件元数据到 MySQL，并将路径缓存到 Redis。

- 请求示例:

  - Headers:
    - `Content-Type: multipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  - Form 表单:
    - `avatar`（必填）：头像文件

- 返回示例:

  - 成功:

    ```
    json复制编辑{
      "code": 0,
      "msg": "upload success"
    }
    ```

  - 错误 (如文件格式错误):

    ```
    json复制编辑{
      "code": 2001,
      "msg": "avatar verify failed(.jpg)"
    }
    ```

------

### 6. 上传文档

- **接口地址**: `POST /upload/document`

- **功能说明**:
  上传文档接口，支持上传 `.docx` 格式的文档。接口可接收 `visibility` 参数，用于设置文档的可见性（默认 `public`，也可以是 `private` 或 `restricted`）。

- 请求示例:

  - Headers:
    - `Content-Type: multipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  - Form 表单:
    - `document`（必填）：文档文件
    - `visibility`（可选）：例如 `public`、`private`、`restricted`

- 返回示例:

  - 成功:

    ```
    json复制编辑{
      "code": 0,
      "msg": "document uploaded successfully"
    }
    ```

  - 错误 (如文件格式错误):

    ```
    json复制编辑{
      "code": 3001,
      "msg": "document verify failed (.docx)"
    }
    ```

------

### 7. 删除文档

- **接口地址**: `DELETE /delete/document`

- **功能说明**:
  根据提供的文件名删除指定文档。若当前用户为文档所有者，则会从腾讯云 COS 删除文件，同时删除数据库中的文件元数据和 Redis 缓存记录。

- 请求示例:

  - Headers:
    - `Authorization: Bearer <jwt_token>`
  - Query 参数:
    - `filename`（必填）：文档文件名（包括扩展名），例如 `/delete/document?filename=document1.docx`

- 返回示例:

  - 成功:

    ```
    json复制编辑{
      "code": 0,
      "msg": "file deleted successfully"
    }
    ```

  - 错误 (如缺少文件名):

    ```
    json复制编辑{
      "code": 4001,
      "msg": "filename is required"
    }
    ```

------

### 8. 更新文档

- **接口地址**: `PUT /update/document`

- **功能说明**:
  更新文档接口，通过提供旧文件名（`filename`）和新文档文件（及可选的新文件名 `newfilename`），更新指定文档。接口会先验证修改权限，删除旧文件（从 COS 中），再上传新文件，并更新数据库和缓存记录。

- 请求示例:

  - Headers:

    - `Content-Type: multipart/form-data`
    - `Authorization: Bearer <jwt_token>`

  - Query 参数:

    - `filename`（必填）：原文档文件名，例如 `document1.docx`
    - `newfilename`（可选）：新文件名，如不提供则使用原文件名

  - Form 表单:

    - `document`（必填）：新的文档文件（仅支持 `.docx`）

  - 示例 URL:

    ```
    bash
    复制编辑
    /update/document?filename=document1.docx&newfilename=document2.docx
    ```

- 返回示例:

  - 成功:

    ```
    json复制编辑{
      "code": 0,
      "msg": "file updated successfully"
    }
    ```

  - 错误 (如权限不足):

    ```
    json复制编辑{
      "code": 5001,
      "msg": "User CannotChange file"
    }
    ```

------

### 9. 获取文档

- **接口地址**: `GET /get/document`

- **功能说明**:
  根据提供的文件名获取文档的访问 URL。接口会先检查用户对该文件的访问权限，再返回存储在腾讯云 COS 上的文档 URL。

- 请求示例:

  - Headers:
    - `Authorization: Bearer <jwt_token>`
  - Query 参数:
    - `filename`（必填）：文档文件名，例如 `/get/document?filename=document1.docx`

- 返回示例:

  - 成功:

    ```
    json复制编辑{
      "code": 0,
      "msg": "document get successfully",
      "URL": "https://<your-cos-domain>/documents/alice/document1.docx"
    }
    ```

  - 错误 (如缺少文件名):

    ```
    json复制编辑{
      "code": 6001,
      "msg": "filename is required"
    }
    ```

------

## 认证说明

对于所有需要 JWT 认证的接口，请在请求头中添加如下内容：

```
makefile


复制编辑
Authorization: Bearer <jwt_token>
```

JWT 令牌可通过 [用户登录](#用户登录) 接口获取。认证中间件会验证该令牌，并将验证后的用户名注入到请求上下文中，以便后续接口进行权限判断。

------

## 错误码及注意事项

- **返回字段说明**:
  - `code`: 错误码。`0` 表示成功，其它数值表示相应错误，具体错误码定义见项目中的 `consts` 包。
  - `msg`: 返回信息，描述成功或错误的详细信息。
- **常见错误码**:
  - `ShouldBindFailed`: 参数绑定失败
  - `UserAlreadyExist`: 用户已存在
  - `PasswordHashedWrong`: 密码加密失败
  - `MysqlQueryFailed`: MySQL 查询错误
  - `RedisQueryFailed`: Redis 操作错误
  - 以及文档操作中可能遇到的其它错误码（const.go中）
- **文件格式要求**:
  - 头像上传仅支持 `.jpg` 格式
  - 文档上传仅支持 `.docx` 格式
- **存储说明**:
  - 文件最终存储于腾讯云 COS，文件元数据存储在 MySQL，并在 Redis 中缓存相关路径信息。
- **其他注意事项**:
  - 上传和更新文件时，请确保用户已通过 JWT 认证，接口会根据当前用户信息进行权限校验。
  - 对于 WebSocket 连接，建议在前端做好连接异常和重连处理。


## 备注
写的特别💩。
确实有很多东西没有实现，比如权限管理，比如WebSocket深入（同步等），还有能docker部署，远程方法调用这些，遗憾很多。然后就是了解了Elasticsearch，感觉比COS更好用。

