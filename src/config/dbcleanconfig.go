package config

import (
	"github.com/SongZihuan/ssh-watcher/src/utils"
	"time"
)

type DBCleanConfig struct {
	ExecutionIntervalHour        int64         `yaml:"execution-interval-hour"`
	SSHRecordSaveRetentionPeriod string        `yaml:"ssh-record-save-retention-period"`
	SSHRecordSaveTime            time.Duration `yaml:"-"`
}

func (d *DBCleanConfig) setDefault() {
	if d.ExecutionIntervalHour <= 0 {
		d.ExecutionIntervalHour = 6
	}

	if d.SSHRecordSaveRetentionPeriod == "" {
		d.SSHRecordSaveRetentionPeriod = "3M"
	}

	return
}

func (d *DBCleanConfig) check() (err ConfigError) {
	d.SSHRecordSaveTime = utils.ReadTimeDuration(d.SSHRecordSaveRetentionPeriod)

	if d.SSHRecordSaveTime == 0 {
		return NewConfigError("bad ssh-record-save-retention-period")
	}

	if d.SSHRecordSaveTime == -1 {
		_ = NewConfigWarning("ssh-record-save-retention-period is set to be saved permanently")
	} else if d.SSHRecordSaveTime < time.Minute*5 {
		return NewConfigError("bad ssh-record-save-retention-period, must more than 5 minute")
	}

	return nil
}
