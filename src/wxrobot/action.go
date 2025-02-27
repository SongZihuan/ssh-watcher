package wxrobot

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

	logger.Errorf("WxRobot Send Error: %s", err.Error())
}

func SendStart() {
	logError(Send("服务启动完成。", true))
}

func SendWaitStop(reason string) {
	if reason == "" {
		reason = "无。"
	} else if !strings.HasSuffix(reason, "。") {
		reason += "。"
	}

	logError(Send(fmt.Sprintf("服务即将停止。原因：%s", reason), true))
}

func SendStop(exitcode int, numGoroutine int) {
	logError(Send(fmt.Sprintf("服务停止。退出代码：%d。剩余协程数：%d", exitcode, numGoroutine), true))
}

func SendSshBanned(ip string, loc *apiip.QueryIpLocationData, to string, reason string) {
	if reason == "" {
		reason = "无。"
	} else if !strings.HasSuffix(reason, "。") {
		reason += "。"
	}

	if loc == nil {
		logError(Send(fmt.Sprintf("IP %s （无定位信息） 连接到 %s 被拒。原因：%s", ip, to, reason), true))
	} else {
		logError(Send(fmt.Sprintf("IP %s （%s） 连接到 %s 被拒。原因：%s", ip, loc.String(), to, reason), true))
	}

}

func SendSshSuccess(ip string, loc *apiip.QueryIpLocationData, to string, mark string) {
	if mark == "" {
		mark = "无。"
	} else if !strings.HasSuffix(mark, "。") {
		mark += "。"
	}

	if loc == nil {
		logError(Send(fmt.Sprintf("IP %s （无定位信息） 连接到 %s 成功。备注：%s", ip, to, mark), false))
	} else {
		logError(Send(fmt.Sprintf("IP %s （%s） 连接到 %s 成功。备注：%s", ip, loc.String(), to, mark), false))
	}
}
