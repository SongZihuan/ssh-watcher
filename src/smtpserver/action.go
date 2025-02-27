package smtpserver

import (
	"fmt"
	"github.com/SongZihuan/ssh-watcher/src/api/apiip"
	"github.com/SongZihuan/ssh-watcher/src/logger"
	"strings"
)

func logError(err error) {
	if err == nil {
		return
	}

	logger.Errorf("SMTP Send Error: %s", err.Error())
}

func SendStart() {
	logError(Send("服务启动完成", "服务启动/重启完成。"))
}

func SendWaitStop(reason string) {
	if reason == "" {
		reason = "无。"
	} else if !strings.HasSuffix(reason, "。") {
		reason += "。"
	}

	logError(Send("服务停止", fmt.Sprintf("服务即将停止。原因：%s", reason)))
}

func SendStop(exitcode int, numGoroutine int) {
	logError(Send("服务停止", fmt.Sprintf("服务停止。退出代码：%d。剩余协程数：%d。", exitcode, numGoroutine)))
}

func SendSshBanned(ip string, loc *apiip.QueryIpLocationData, to string, reason string) {
	if reason == "" {
		reason = "无。"
	} else if !strings.HasSuffix(reason, "。") {
		reason += "。"
	}

	if loc == nil {
		logError(Send("SSH异常请求（拒绝）", fmt.Sprintf("IP %s （无定位信息） 连接到 %s 被拒。原因：%s", ip, to, reason)))
	} else {
		logError(Send("SSH异常请求（拒绝）", fmt.Sprintf("IP %s （%s） 连接到 %s 被拒。原因：%s", ip, loc.String(), to, reason)))
	}
}

func SendSshSuccess(ip string, loc *apiip.QueryIpLocationData, to string, mark string) {
	if mark == "" {
		mark = "无。"
	} else if !strings.HasSuffix(mark, "。") {
		mark += "。"
	}

	if loc == nil {
		logError(Send("SSH请求（通过）", fmt.Sprintf("IP %s （无定位信息） 连接到 %s 成功。备注：%s", ip, to, mark)))
	} else {
		logError(Send("SSH请求（通过）", fmt.Sprintf("IP %s （%s） 连接到 %s 成功。备注：%s", ip, loc.String(), to, mark)))
	}
}
