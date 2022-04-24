# API Document

---

## Common

Only `POST` requests are allowed.

For every request, you should have `Authorization` on its header.

Authorization header is a string like `Basic <token>`(basic auth).

```
Authorization: Basic <base64('username:password')>
```

---

## Response

All responses are JSON encoded.

| code | meaning                   |
|------|---------------------------|
| -1   | invalid or missing params |
| 0    | success                   |
| 1    | failure and msg are given |

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

### List devices: `/device/list`

Parameters: **None**

The `id` of device is persistent, its length always equals 64. It's unique for every device and won't change, so you should identify every device by this.

The key of the device object is its connection UUID, it's random and temporary.

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

### Basic operations: `/device/:act`

Parameters: `:act` and `device` (device ID)

The `:act` could be `lock`, `logoff`, `hibernate`, `suspend`, `restart`, `shutdown` and `offline`.

For example, when you call `/device/restart`, your device will restart.

```
{
    "code": 0
}
```

---

### Take screenshot: `/device/screenshot/get`

Parameters: `device` (device ID)

If screenshot is captured successfully, it gives you the image directly. If failed, then the following response are given.

```
{
    "code": 1,
    "msg": "${i18n|noDisplayFound}"
}
```

---

### Get a file: `/device/file/get`

Parameters: `file` (path to file) and `device` (device ID)

If file exists and is accessible, then the file is given directly. If failed, then the following response are given.

```
{
    "code": 1,
    "msg": "${i18n|fileOrDirNotExist}"
}
```

---

### Delete a file: `/device/file/remove`

Parameters: `file` (path to file) and `device` (device ID)

If file exists and is deleted successfully, then `code` will be `0`.

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

### List files: `/device/file/list`

Parameters: `path` (folder to be listed) and `device` (device ID)

If `path` is empty, then it gives you volumes list (windows) or gives files on `/`.

`type` `0` means file, `1` means folder and `2` means volume (windows).

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

### List processes: `/device/process/list`

Parameters: `device` (device ID)

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

### Kill a process: `/device/process/kill`

Parameters: `pid` and `device` (device ID)

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
