package sshwatcher

import (
	"errors"
	"fmt"
	"github.com/SongZihuan/ssh-watcher/src/config"
	"github.com/SongZihuan/ssh-watcher/src/database"
	"github.com/SongZihuan/ssh-watcher/src/flagparser"
	"github.com/SongZihuan/ssh-watcher/src/ipcheck"
	"github.com/SongZihuan/ssh-watcher/src/logger"
	"github.com/SongZihuan/ssh-watcher/src/notify"
	"github.com/SongZihuan/ssh-watcher/src/redisserver"
	"github.com/SongZihuan/ssh-watcher/src/smtpserver"
	"github.com/SongZihuan/ssh-watcher/src/sshserver"
	"github.com/SongZihuan/ssh-watcher/src/utils"
	"os"
	"sync"
	"time"
)

func MainV1() (exitcode int) {
	var err error

	if ipcheck.SupportIPv4() {
		fmt.Println("Server support ipv4.")
	} else {
		fmt.Println("Server dosen't support ipv4.")
	}

	if ipcheck.SupportIPv6() {
		fmt.Println("Server support ipv6.")
	} else {
		fmt.Println("Server dosen't support ipv6.")
	}

	if !ipcheck.SupportIPv4() && !ipcheck.SupportIPv6() {
		fmt.Println("Server doesn't support ipv4 and ipv6.")
		return 1
	}

	defer func() {
		// 此处使用同步通知
		notify.SyncSendStop(exitcode)
	}()

	err = flagparser.InitFlag()
	if errors.Is(err, flagparser.StopFlag) {
		return 0
	} else if err != nil {
		return utils.ExitByError(err)
	}

	if !flagparser.IsReady() {
		return utils.ExitByErrorMsg("flag parser unknown error")
	}

	utils.SayHellof("%s", "The backend service program starts normally, thank you.")
	defer func() {
		if exitcode != 0 {
			utils.SayGoodByef("%s", "The backend service program is offline/shutdown with error, thank you.")
		} else {
			utils.SayGoodByef("%s", "The backend service program is offline/shutdown normally, thank you.")
		}
	}()

	cfgErr := config.InitConfig(flagparser.ConfigFile())
	if cfgErr != nil && cfgErr.IsError() {
		return utils.ExitByError(cfgErr)
	}

	if !config.IsReady() {
		return utils.ExitByErrorMsg("config parser unknown error")
	}

	err = logger.InitLogger(os.Stdout, os.Stderr)
	if err != nil {
		return utils.ExitByError(err)
	}

	if !logger.IsReady() {
		return utils.ExitByErrorMsg("logger unknown error")
	}

	err = smtpserver.InitSmtp()
	if err != nil {
		logger.Errorf("init smtp fail: %s", err.Error())
		return 1
	}

	err = database.InitSQLite()
	if err != nil {
		logger.Errorf("init sqlite fail: %s", err.Error())
		return 1
	}
	defer database.CloseSQLite()

	cleaner, err := database.NewCleaner()
	if err != nil {
		logger.Errorf("create sqlclear fail: %s", err.Error())
		return 1
	}

	err = cleaner.Start()
	if err != nil {
		logger.Errorf("start sqlclear fail: %s", err.Error())
		return 1
	}
	defer func() {
		_ = cleaner.Stop()
	}()

	err = redisserver.InitRedis()
	if err != nil {
		logger.Errorf("init redis fail: %s\n", err.Error())
		return 1
	}
	defer redisserver.CloseRedis()

	ser, err := sshserver.NewSshServer(&config.GetConfig().SSH.Forward)
	if err != nil {
		logger.Errorf("init ssh watcher server fail: %s\n", err.Error())
		return 1
	}

	logger.Executablef("%s", "ready")
	logger.Infof("run mode: %s", config.GetConfig().GlobalConfig.GetRunMode())

	err = ser.Start()
	if err != nil {
		logger.Errorf("start ssh watcher server failed: %s\n", err.Error())
		return 1
	}
	defer func() {
		_ = ser.Stop()
	}()

	notify.SendStart() // 此处是Start不是WaitStart

	select {
	case <-config.GetSignalChan():
		notify.SendWaitStop("接收到退出信号")

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()

			_ = ser.Stop() // 提前关闭，同时代码上面的 defer 兜底
		}()

		go func() {
			defer wg.Done()

			_ = cleaner.Stop() // 提前关闭，同时代码上面的 defer 兜底
		}()

		wg.Wait()

		time.Sleep(1 * time.Second)
		return 0
	}
	// 无法抵达
}
