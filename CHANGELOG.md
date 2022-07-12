## v0.1.4

* Add: desktop viewer (experimental).
* Optimize: project structure.

* 新增：桌面监控(试验版)。
* 优化：项目结构。



## v0.1.3

* Optimize: basic operations for macOS.
* Fix: deadlock when download more than one item.

* 优化：macOS下，基础操作改为API调用的方式实现。
* 修复：下载目录或多文件时发生死锁，导致压缩文件不完整。



## v0.1.2

* Optimize: compress frontend assets.

* 优化：压缩前端资源，加快加载速度。



## v0.1.1

* Add: text file editor.
* Add: explorer multi-select.
* Add: explorer search.
* Fix: some potential bugs.
* BREAKING-CHANGE: API `/device/file/get` parameter `file` changed to `files`.
* BREAKING-CHANGE: API `/device/file/remove` parameter `file` changed to `files`.

* 新增：文本文件编辑器。
* 新增：文件管理器多选。
* 新增：文件管理器过滤。
* 修复：一些潜在的bug。
* 破坏性变动：API `/device/file/get` 参数 `file` 变为 `files`。
* 破坏性变动：API `/device/file/remove` 参数 `file` 变为 `files`。



## v0.1.0

* fix: don't refresh after file upload.
* fix: don't display error when screenshot fails.

* 修复：文件上传成功后文件管理器不会自动刷新。
* 修复：截图失败时不会显示错误提示。



## v0.0.9

* Optimize: performance of front-end and back-end.
* Optimize: security vulnerability.

* 优化：前后端性能。
* 优化：安全问题。



## v0.0.8

* Add: file upload.
* Optimize: project structure.

* 新增：文件上传功能。
* 优化：项目结构。



## v0.0.7

* Add: detail info tooltip of cpu, ram and disk.

* 新增：CPU、内存、磁盘的详细信息的提示。



## v0.0.6

* Update: i18n.
* Remove: initial columns state.

* 更新：国际化。
* 移除：默认隐藏部分信息。



## v0.0.5

* Add: server and client now support macOS.
* Add: shutdown and reboot on macOS (root permission required).
* Update: pty are used on Unix-like OS to provide a full-functional terminal.
* Update: improved the support of terminal on Windows and fixed some bugs.

* 新增：服务端和客户端已支持macOS系统。
* 新增：macOS现在将支持关机和重启功能（需要root权限）。
* 更新：类unix系统的终端现已改用pty实现，以提供完整的终端功能。
* 更新：改进了windows下的终端表现，修复了一些bug。



## v0.0.4

* Add: i18n support.
* Note: from this version, you just need to upgrade server manually and client will automatically self upgrade.

* 新增：支持中英文国际化。
* 注意：从本版本开始，只需要更新服务端即可，客户端会自动完成更新。



## v0.0.3

* Add: network IO speed monitoring.
* Add: support client self-upgrade.
* Fix: garbled characters when display Chinese on Unix-like OS.
* BREAKING-CHANGE: module `Device` has changed.
* THIS RELEASE IS **NOT** COMPATIBLE WITH LAST RELEASE.

* 新增：网络IO速度监控。
* 新增：客户端自行升级。
* 修复：在类Unix系统中使用terminal时中文乱码。
* 破坏性变动：`Device`类型已更改。
* 本版本**不**兼容上一版本，暂时仍需要手动升级客户端。



## v0.0.2

* Add: latency check.
* Add: progress bar of cpu usage, memory usage and disk usage.
* BREAKING-CHANGE: module `Device` has changed.
* THIS RELEASE IS **NOT** COMPATIBLE WITH LAST RELEASE.

* 新增：网络延迟检测。
* 新增：CPU使用率进度、内存使用进度、硬盘使用进度。
* 破坏性变动：`Device`类型已更改。
* 本版本**不**兼容上一版本，暂时仍需要手动升级客户端。



## v0.0.1

* First release.

* 这是第一个发行版。