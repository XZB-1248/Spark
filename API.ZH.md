# API 文档

---

## 通用

所有请求均为`POST`。

每次请求都必须在Header中带上`Authorization`。

`Authorization`请求头格式：`Basic <token>`（basic auth）。

```
Authorization: Basic <base64('username:password')>
```

---

## 响应

所有响应均是JSON格式。

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

设备的`id`是一串64位的字符串，每台设备独一无二，一般不会变化。识别设备主要靠这个。下文中提到的设备ID也指的是这个。

每个device对象所对应的key，是它的本次连接的连接ID，这个ID是随机、临时的，每次重连就会变化，不建议使用。

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
                "usage": 8.658854166666668
            },
            "mem": {
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

### 获取截屏：`/device/screenshot/get`

参数：`device`（设备ID）

如果截屏获取成功，则会直接以图片的形式输出。如果截屏失败，如下响应会被输出（错误信息不止这一个）。

```
{
    "code": 1,
    "msg": "${i18n|noDisplayFound}"
}
```

---

### 读取设备上的文件：`/device/file/get`

参数：`file`（文件路径） 以及 `device`（设备ID）

如果文件存在且可访问，则文件会直接输出。否则，会给出以下响应。

```
{
    "code": 1,
    "msg": "${i18n|fileOrDirNotExist}"
}
```

---

### 删除设备上的文件：`/device/file/remove`

参数：`file`（文件路径） 以及 `device`（设备ID）

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

### 列举设备上的文件和目录：`/device/file/list`

参数：`path`（父目录路径） 以及 `device`（设备ID）

如果`path`为空，windows下会给出磁盘列表，其它系统会默认输出`/`目录下的文件和目录。

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
