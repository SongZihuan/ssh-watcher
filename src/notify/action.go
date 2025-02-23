package notify

import (
	"github.com/SongZihuan/ssh-watcher/src/api/apiip"
	"github.com/SongZihuan/ssh-watcher/src/config"
	"github.com/SongZihuan/ssh-watcher/src/smtpserver"
	"github.com/SongZihuan/ssh-watcher/src/wxrobot"
	"runtime"
	"sync"
)

var hasSendStart = false

func SendStart() {
	if !config.IsReady() {
		panic("config is not ready")
	} else if config.GetConfig().Quite.IsEnable(false) {
		return
	}

	go wxrobot.SendStart()
	go smtpserver.SendStart()

	hasSendStart = true
}

func SendWaitStop(reason string) {
	if !config.IsReady() {
		panic("config is not ready")
	} else if config.GetConfig().Quite.IsEnable(false) {
		return
	}

	go wxrobot.SendWaitStop(reason)
	go smtpserver.SendWaitStop(reason)
}

func AsyncSendStop(exitcode int) {
	if !config.IsReady() {
		panic("config is not ready")
	} else if config.GetConfig().Quite.IsEnable(false) {
		return
	}

	if !hasSendStart {
		return
	}

	numGoroutine := runtime.NumGoroutine()

	go wxrobot.SendStop(exitcode, numGoroutine)
	go smtpserver.SendStop(exitcode, numGoroutine)
}

func SyncSendStop(exitcode int) {
	if !config.IsReady() {
		panic("config is not ready")
	} else if config.GetConfig().Quite.IsEnable(false) {
		return
	}

	if !hasSendStart {
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	numGoroutine := runtime.NumGoroutine()

	go func() {
		defer wg.Done()
		wxrobot.SendStop(exitcode, numGoroutine)
	}()

	go func() {
		defer wg.Done()
		smtpserver.SendStop(exitcode, numGoroutine)
	}()

	wg.Wait()
}

func SendSshBanned(ip string, loc *apiip.QueryIpLocationData, to string, reason string) {
	if !config.IsReady() {
		panic("config is not ready")
	} else if config.GetConfig().Quite.IsEnable(false) {
		return
	}

	go wxrobot.SendSshBanned(ip, loc, to, reason)
	go smtpserver.SendSshBanned(ip, loc, to, reason)
}

func SendSshSuccess(ip string, loc *apiip.QueryIpLocationData, to string, mark string) {
	if !config.IsReady() {
		panic("config is not ready")
	} else if config.GetConfig().Quite.IsEnable(false) {
		return
	}

	go wxrobot.SendSshSuccess(ip, loc, to, mark)
	go smtpserver.SendSshSuccess(ip, loc, to, mark)
}
