# API 文档

---

## 通用

所有请求均为`POST`。

### 鉴权

每次请求都必须在Header中带上`Authorization`。
<br />
`Authorization`请求头格式：`Basic <token>`（basic auth）。

```
Authorization: Basic <base64('username:password')>
```
例如：
```
Authorization: Basic WFpCOjEyNDg=
```

在最初的Basic Authentication之后，服务端会分配一个`Authorization`的Cookie。
<br />
该Cookie可用于请求的后续鉴权，可以不再附带Authorization头。

---

## 响应

所有响应均是JSON格式。
<br />
`code` 有三种结果，分别为`-1`，`0`和`1`，含义如下。

| code | meaning    |
|------|------------|
| -1   | 参数缺失或无效    |
| 0    | 成功         |
| 1    | 失败，并输出错误信息 |

```
{
    "code": -1,
    "msg": "${i18n|invalidParameter}"
}
```
```
{
    "code": 0,
    "data": {
        ...
    }
}
```
```
{
    "code": 1,
    "msg": "${i18n|deviceNotExists}"
}
```

---

### 获取设备列表：`/device/list`

参数：**无**

设备的`id`是一串64位的字符串，每台设备独一无二，一般不会变化。
<br />
识别设备主要靠这个。下文中提到的设备ID也指的是这个。
<br />
每个device对象所对应的key，是它的本次连接的连接ID。
<br />
连接ID是随机、临时的，每次重连就会变化，不建议使用。

```
{
    "code": 0,
    "data": {
        "1de601ca-7738-4b77-a081-57d3fc9c4482": {
            "id": "1a23e7660cde01285ca241d5f5d3cf2c5bc39e02c1df7a30b58fbde2938b0375",
            "os": "windows",
            "arch": "amd64",
            "lan": "192.168.1.1",
            "wan": "1.1.1.1",
            "mac": "00:00:00:00:00:00",
            "net": {
                "sent": 0,
                "recv": 60
            },
            "cpu": {
                "model": "Intel(R) Core(TM) i5-9300H CPU @ 2.40GHz",
                "usage": 8.658854166666668,
                "cores": {
                    "logical": 8,
                    "physical": 4
                }
            },
            "ram": {
                "total": 8432967680,
                "used": 5109829632,
                "usage": 60.593492420452385
            },
            "disk": {
                "total": 1373932810240,
                "used": 185675567104,
                "usage": 13.51416646579435
            },
            "uptime": 1015,
            "latency": 10,
            "hostname": "LOCALHOST",
            "username": "EXAMPLE"
        }
    }
}
```

---

### 基础操作：`/device/:act`

参数：`:act` 以及 `device`（设备ID）

参数 `:act` 可以是这些： `lock`，`logoff`，`hibernate`，`suspend`，`restart`，`shutdown` 以及 `offline`。

例如，如果你调用 `/device/restart`，那对应设备就会重启。

```
{
    "code": 0
}
```

---

### 执行命令：`/device/exec`

参数：`cmd`、`args`以及`device`（设备ID）

示例:
```http request
POST http://localhost:8000/api/device/exec HTTP/1.1
Host: localhost:8000
Content-Length: 116
Content-Type: application/x-www-form-urlencoded
User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.64 Safari/537.36 Edg/101.0.1210.47
Origin: http://localhost:8000
Referer: http://localhost:8000/

cmd=taskkill&args=%2Ff%20%2Fim%20regedit.exe&device=bc7e49f8f794f80ffb0032a4ba516c86d76041bf2023e1be6c5dda3b1ee0cf4c
```

```
{
    "code": 0
}
```

---

### 获取截屏：`/device/screenshot/get`

参数：`device`（设备ID）

如果截屏获取成功，则会直接以图片的形式输出。
<br />
如果截屏失败，如下响应会被输出（错误信息不唯一）。

```
{
    "code": 1,
    "msg": "${i18n|noDisplayFound}"
}
```

---

### 读取设备上的文件：`/device/file/get`

参数：`files`（文件数组） 以及 `device`（设备ID）

如果文件存在且可访问，则文件会直接输出。
<br />
否则，会给出错误原因。
<br />
如果`files`为文件数组或者目录，则会输出一个zip文件。

```
{
    "code": 1,
    "msg": "${i18n|fileOrDirNotExist}"
}
```

---

### 删除设备上的文件：`/device/file/remove`

参数：`files`（文件数组） 以及 `device`（设备ID）

如果文件存在且被成功删除，则`code`为`0`。

```
{
    "code": 0
}
```
```
{
    "code": 1,
    "msg": "${i18n|fileOrDirNotExist}"
}
```

---

### 上传文件到目录：`/device/file/upload`

**GET**参数：`file`（文件名）、`path`（路径）和`device`（设备ID）

文件内容需要作为**请求体body**发送。
<br />
**请求体body**中的任何内容都会被写到指定地文件中。
<br />
如果存在同名文件，则会被**覆盖**！

示例:
```http request
POST http://localhost:8000/api/device/file/upload?path=D%3A%5C&file=Test.txt&device=bc7e49f8f794f80ffb0032a4ba516c86d76041bf2023e1be6c5dda3b1ee0cf4c HTTP/1.1
Host: localhost:8000
Content-Length: 12
Content-Type: application/octet-stream
User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.64 Safari/537.36 Edg/101.0.1210.47
Origin: http://localhost:8000
Referer: http://localhost:8000/

Hello World.
```

如果文件上传成功，则`code`为`0`。
<br />
文件`D:\Test.txt`会写入：`Hello World.`。

```
{
    "code": 0
}
```
```
{
    "code": 1,
    "msg": "${i18n|fileOrDirNotExist}"
}
```

---

### 列举设备上的文件和目录：`/device/file/list`

参数：`path`（父目录路径） 以及 `device`（设备ID）

如果`path`为空，windows下会给出磁盘列表。
<br />
其它系统会默认输出`/`目录下的文件和目录。

`type`有三种结果：`0`代表文件，`1`代表目录，`2`代表磁盘（windows）。

```
{
    "code": 0,
    "data": {
        "files": [
            {
                "name": "home",
                "size": 4096,
                "time": 1629627926,
                "type": 1
            },
            {
                "name": "Spark",
                "size": 8192,
                "time": 1629627926,
                "type": 0
            }
        ]
    }
}
```
```
{
    "code": 1,
    "msg": "${i18n|fileOrDirNotExist}"
}
```

---

### 获取进程列表`/device/process/list`

参数：`device`（设备ID）

```
{
    "code": 0,
    "data": {
        "processes": [
            {
                "name": "[System Process]",
                "pid": 0
            },
            {
                "name": "System",
                "pid": 4
            },
            {
                "name": "Registry",
                "pid": 124
            },
            {
                "name": "smss.exe",
                "pid": 392
            },
            {
                "name": "winlogon.exe",
                "pid": 456
            }
        ]
    }
}
```

---

### 结束进程：`/device/process/kill`

参数：`pid` 以及 `device`（设备ID）

```
{
    "code": 0
}
```
```
{
    "code": 1,
    "msg": "${i18n|deviceNotExists}"
}
```
