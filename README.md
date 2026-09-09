# cat-led

懒猫的灯很漂亮，但是晚上开着太刺眼了，影响睡眠。所以开发了这款应用，让懒猫设备灯自动根据您的设置进行开启和关闭。

懒猫微服商店

![image-20250331161759893](https://lzc-playground-1301583638.cos.ap-chengdu.myqcloud.com/guidelines/395/20250331161800070.png?imageSlim)

PC端![image-20250331161932400](https://lzc-playground-1301583638.cos.ap-chengdu.myqcloud.com/guidelines/395/20250331161932589.png?imageSlim)

懒猫微服开发者：[欢迎使用懒猫微服 | 懒猫微服开发者手册](https://developer.lazycat.cloud/)

懒猫微服官网：[懒猫微服](https://lazycat.cloud/)

## 0.2.0

- 灯光样式在固定展示区内平滑切换，优化暖色光晕与液态光效，支持减少动态效果。
- 灯具操作以设备返回状态为准，避免重复请求；样式保存失败可直接重试。
- 启动器根据设备灯的开关状态显示对应图标，最低系统版本为 lzcos 1.6.2。

两张状态图片位于 `internal/pkg/launchericon/`，通过 Go embed 编入程序。启动读取设备状态、手动开关、定时任务及页面状态刷新后，会按需原子更新 `/lzcapp/run/launcher-icon/icon.png`。此目录由系统管理，重启后由程序重新生成图标；写入失败只记录日志，不回滚灯光操作。

验证：`go test -race ./...`，`./build.sh`。发布 tag 使用 `v0.2.0`，GitHub Actions 沿用现有流程构建并发布 LPK。
