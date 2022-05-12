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