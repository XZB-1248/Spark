# API Document

---

## Common

Only `POST` requests are allowed.

### Authenticate

For every request, you should have `Authorization` on its header.
<br />
Authorization header is a string like `Basic <token>`(basic auth).

```
Authorization: Basic <base64('username:password')>
```
Example:
```
Authorization: Basic WFpCOjEyNDg=
```

After basic authentication, server will assign you an `Authorization` cookie.
<br />
You can use this token cookie to authenticate rest of your requests.

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
    "msg": "${i18n|COMMON.INVALID_PARAMETER}"
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
    "msg": "${i18n|COMMON.DEVICE_NOT_EXIST}"
}
```

---

### List devices: `/device/list`

Parameters: **None**

The `id` of device is persistent, its length always equals 64.
<br />
It's unique for every device and won't change.
<br />
You're recommend to recognize your device by device ID.
<br />
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

### Execute command: `/device/exec`

Parameters: `cmd`, `args` and `device` (device ID)

Example:
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

### Take screenshot: `/device/screenshot/get`

Parameters: `device` (device ID)

If screenshot is captured successfully, it gives you the image directly.
<br />
If failed, then the following response are given.

```
{
    "code": 1,
    "msg": "${i18n|DESKTOP.NO_DISPLAY_FOUND}"
}
```

---

### Get files: `/device/file/get`

Parameters: `files` (array of files) and `device` (device ID)

If files exist and are accessible, then the archive file or file itself is given directly.
<br />
If unable to read files, then the following response are given.
<br />
A zip file is given if multiple files (including directory) are given.

```
{
    "code": 1,
    "msg": "${i18n|EXPLORER.FILE_OR_DIR_NOT_EXIST}"
}
```

---

### Delete files: `/device/file/remove`

Parameters: `files` (array of files) and `device` (device ID)

If files exist and are deleted successfully, then `code` will be `0`.

```
{
    "code": 0
}
```
```
{
    "code": 1,
    "msg": "${i18n|EXPLORER.FILE_OR_DIR_NOT_EXIST}"
}
```

---

### Upload file: `/device/file/upload`

**Query Parameters**: `file` (file name), `path` and `device` (device ID)

File itself should be sent in the request **body**.
<br />
**Anything** represented in the request **body** will be saved to the device.
<br />
If same file exists, then it will be **overwritten**.

Example:
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

If file uploaded successfully, then `code` will be `0`.
<br />
And `D:\Test.txt` will be created with the content of `Hello World.`.

```
{
    "code": 0
}
```
```
{
    "code": 1,
    "msg": "${i18n|EXPLORER.FILE_OR_DIR_NOT_EXIST}"
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
    "msg": "${i18n|EXPLORER.FILE_OR_DIR_NOT_EXIST}"
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
    "msg": "${i18n|COMMON.DEVICE_NOT_EXIST}"
}
```
