# cat-led

懒猫的灯很漂亮，但是晚上开着太刺眼了，影响睡眠。所以开发了这款应用，让懒猫设备灯自动根据您的设置进行开启和关闭。

懒猫微服商店

![image-20250331161759893](https://lzc-playground-1301583638.cos.ap-chengdu.myqcloud.com/guidelines/395/20250331161800070.png?imageSlim)

PC端![image-20250331161932400](https://lzc-playground-1301583638.cos.ap-chengdu.myqcloud.com/guidelines/395/20250331161932589.png?imageSlim)

懒猫微服开发者：[欢迎使用懒猫微服 | 懒猫微服开发者手册](https://developer.lazycat.cloud/)

懒猫微服官网：[懒猫微服](https://lazycat.cloud/)

## 0.4.4

- 关于窗口默认显示会眨眼、左右看的黑猫，点击可切换为抱着毛线球荡秋千的小猫。
- 新增系统运行时长，以 `1小时2分` 等易读格式显示，支持深浅主题和中英文切换。

## 0.4.3

- 点击标题旁的信息图标，可查看应用版本、作者和项目主页。
- 关于窗口新增入场动效和小猫彩蛋：摸摸小猫，它会睁眼送你小星星；再次点击就继续打盹。

### Application version

`./build.sh` embeds the version from `package.yml` in the binary. Set `VERSION`
to override it for a custom build, for example `VERSION=0.4.4 ./build.sh`.
Plain `go build` or `go run` without linker flags reports `dev`.
Run `./dist/cat-led --version` to check it without starting the server, or click
the info icon beside the main page title to open About. The device version shown
in the user information area is the LazyCat OS version, separate from the app.
Tap the black cat in About to play with a ball of yarn; tap again to switch back.
System uptime comes from Linux `/proc/uptime`, including suspend time, and is
refreshed every minute while About is open. It measures time since system boot,
not the lifetime of the app process. If the clock cannot be read, About shows
an unavailable message. Both cat animations respect reduced-motion settings.

## 0.2.0

- 灯光样式在固定展示区内平滑切换，优化暖色光晕与液态光效，支持减少动态效果。
- 灯具操作以设备返回状态为准，避免重复请求；样式保存失败可直接重试。
- 启动器根据设备灯的开关状态显示对应图标，最低系统版本为 lzcos 1.6.2。

两张状态图片位于 `internal/pkg/launchericon/`，通过 Go embed 编入程序。启动读取设备状态、手动开关、定时任务及页面状态刷新后，会按需原子更新 `/lzcapp/run/launcher-icon/icon.png`。此目录由系统管理，重启后由程序重新生成图标；写入失败只记录日志，不回滚灯光操作。

验证：`go test -race ./...`，`./build.sh`。发布 tag 使用 `v0.2.0`，GitHub Actions 沿用现有流程构建并发布 LPK。

## 0.3.0

- Add a separate main-page power-schedule entry for LazyCat administrators.
- Support one-time and weekly shutdown/wake pairs, IANA timezones, overnight wake,
  cancellation, persistent recovery, and visible device/plan errors.
- Use `github.com/lib-x/rtc v0.1.0` for RTC alarm writes and readback. Shutdown
  uses the existing LazyCat SDK poweroff API.
- Reuse SQLite with a separate `rtc_power_plan` table; no database service is needed.
- Sign OIDC sessions, verify current LazyCat roles, protect power mutations from
  cross-origin requests, and render shared user text safely.

The application remains Linux-only on the LazyCat server; Windows/macOS browsers
can use the interface. The RTC library itself exposes native platform capabilities
separately. This application requires `/dev/rtc0` and a hardware clock maintained
in UTC. The package maps the device into the main `app` container and runs the
scheduler as a background task. Host PID access is not required.

Before using automatic power-on, enable the relevant firmware wake setting and
verify S5 wake on the target machine while keeping power connected. RTC readback
confirms the alarm, not physical wake capability. The application must remain
running until shutdown; reinstall/restart recovery never replays missed shutdowns.
One shared plan is supported per device. Other software must not overwrite its
hardware alarm. Single-use wake targets are limited to the next 28 days; driver
limits can be shorter and are reported as errors.

Old unsigned OIDC cookies are rejected. Existing OIDC users sign in again after
upgrade or process restart. The service binds to loopback because LazyCat injects
trusted identity headers through its gateway. Do not expose the internal port
through an additional unauthenticated proxy.

Validation: `go test -race ./...`, `go vet ./...`, and `./build.sh`. Browser checks
use a mock backend and never schedule a real shutdown.


## 0.3.1

- Move the administrator-only power button into the existing top toolbar.
- Add independent shutdown and RTC wake switches; show settings only when enabled.
- Display power plans alongside existing tasks, with distinct timer/alarm icons.
- Add a paper-like RTC dialog entrance and reverse exit with reduced-motion support.
- Keep older paired plans compatible and recheck administrator roles before rearming.

Power-off uses LazyCat SDK `Poweroff`. RTC alarm readback cannot verify successful
OS boot or network recovery.


## 0.3.2

- Keep the RTC dialog header and close button visible while its content scrolls.
- Add English/Chinese switching on the main, settings and sign-in pages, with a
  remembered browser preference and localized status, date and error messages.
- Preserve unsaved form input, user-created schedule names and notification templates
  when the display language changes.

Validation also includes `node --test tests/i18n.test.cjs` for locale selection,
placeholder preservation, error translation and message coverage.
