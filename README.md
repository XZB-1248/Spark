#### [English] [[中文]](./README.ZH.md) [[API Document]](./API.md) [[API文档]](./API.ZH.md)

---

<h1 >Spark</h1>

**[Spark](https://github.com/XZB-1248/Spark)** is a free, safe, open-source, web-based, cross-platform, and full-featured RAT (Remote Administration Tool) that allows you to control all your devices via browser anywhere.

✅ **No data collection**: Spark does not collect any user information.  
✅ **No auto-updates**: The server will not update itself.  
✅ **Direct communication**: Clients communicate exclusively with your server.

---



| ![GitHub repo size](https://img.shields.io/github/repo-size/XZB-1248/Spark?style=flat-square) | ![GitHub issues](https://img.shields.io/github/issues/XZB-1248/Spark?style=flat-square) | ![GitHub closed issues](https://img.shields.io/github/issues-closed/XZB-1248/Spark?style=flat-square) |
|-----------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------|

| [![GitHub downloads](https://img.shields.io/github/downloads/XZB-1248/Spark/total?style=flat-square)](https://github.com/XZB-1248/Spark/releases) | [![GitHub release (latest by date)](https://img.shields.io/github/downloads/XZB-1248/Spark/latest/total?style=flat-square)](https://github.com/XZB-1248/Spark/releases/latest) |
|-|-|

---

## ⚠️ Disclaimer

**THIS PROJECT, ITS SOURCE CODE, AND RELEASES SHOULD ONLY BE USED FOR EDUCATIONAL PURPOSES.**

❌ **Illegal usage is strictly prohibited.**  
❌ **Authors and developers are not responsible for any misuse.**  
✅ **Use it at your own risk.**

If you find security vulnerabilities, **do not open an issue**. Contact me immediately via [email](mailto:i@1248.ink).

---

## 🚀 Quick Start

### Binary Execution

1. Download the executable from the [releases](https://github.com/XZB-1248/Spark/releases) page.
2. Follow the [Configuration](#configuration) instructions.
3. Run the executable and access the web interface at `http://IP:Port`.
4. Generate a client and run it on the target device.
5. Start managing your devices!

---

## ⚙️ Configuration

The configuration file `config.json` should be in the same directory as the executable.

**Example:**

```json
{
    "listen": ":8000",
    "salt": "123456abcdef123456", 
    "auth": {
        "username": "password"
    },
    "log": {
        "level": "info",
        "path": "./logs",
        "days": 7
    }
}
```

### Main Parameters:
- **`listen`** (required): Format `IP:Port`.
- **`salt`** (required): Max length 24 characters. After modification, all clients need to be regenerated.
- **`auth`** (optional): Authentication credentials (`username:password`).
  - Hashed passwords are recommended (`$algorithm$hashed-password`).
  - Supported algorithms: `sha256`, `sha512`, `bcrypt`.
- **`log`** (optional): Logging configuration.
  - `level`: `disable`, `fatal`, `error`, `warn`, `info`, `debug`.
  - `path`: Log directory (default: `./logs`).
  - `days`: Log retention days (default: `7`).

---

## 🛠️ Features

| Feature/OS        | Windows | Linux | MacOS |
|-------------------|---------|-------|-------|
| Process Manager   | ✔       | ✔     | ✔     |
| Kill Process      | ✔       | ✔     | ✔     |
| Network Traffic   | ✔       | ✔     | ✔     |
| File Explorer     | ✔       | ✔     | ✔     |
| File Transfer     | ✔       | ✔     | ✔     |
| File Editor       | ✔       | ✔     | ✔     |
| Delete File       | ✔       | ✔     | ✔     |
| Code Highlighting | ✔       | ✔     | ✔     |
| Desktop Monitor   | ✔       | ✔     | ✔     |
| Screenshot        | ✔       | ✔     | ✔     |
| OS Info           | ✔       | ✔     | ✔     |
| Remote Terminal   | ✔       | ✔     | ✔     |
| * Shutdown        | ✔       | ✔     | ✔     |
| * Reboot          | ✔       | ✔     | ✔     |
| * Log Off         | ✔       | ❌     | ✔     |
| * Sleep           | ✔       | ❌     | ✔     |
| * Hibernate       | ✔       | ❌     | ❌     |
| * Lock Screen     | ✔       | ❌     | ❌     |

🚨 **Functions marked with * may require administrator/root privileges.**

---

## 📸 Screenshots

![overview](./docs/overview.png)  
![terminal](./docs/terminal.png)  
![desktop](./docs/desktop.png)  
![proc_mgr](./docs/procmgr.png)  
![explorer](./docs/explorer.png)  
![overview.cpu](./docs/overview.cpu.png)  
![explorer.editor](./docs/explorer.editor.png)

---

## 🔧 Development

### Components
This project consists of three main components:
- **Client**
- **Server**
- **Front-end**

For OS support beyond Linux and Windows, additional C compilers may be required. For example, to support Android, install [Android NDK](https://developer.android.com/ndk/downloads).

### Build Guide

```bash
# Clone the repository
git clone https://github.com/XZB-1248/Spark
cd ./Spark

# Build the front-end
cd ./web
npm install
npm run build-prod

# Embed static resources
cd ..
go install github.com/rakyll/statik
statik -m -src="./web/dist" -f -dest="./server/embed" -p web -ns web

# Build the client
mkdir ./built
go mod tidy
go mod download
./scripts/build.client.sh

# Build the server
mkdir ./releases
./scripts/build.server.sh
```

---

## 📜 License

Distributed under the [BSD-2 License](./LICENSE).
