package config

import "github.com/SongZihuan/ssh-watcher/src/utils"

type SshRuleListConfig struct {
	RuleList []*SshRuleConfig `yaml:"rules"`

	DefaultBanned       utils.StringBool `yaml:"default-banned"`        // 默认（未名字规则）拒绝连接
	AlwaysAllowIntranet utils.StringBool `yaml:"always-allow-intranet"` // 总是允许内网连接（配置 ip 数据库封禁除外）
	AlwaysAllowLoopback utils.StringBool `yaml:"always-allow-loopback"` // 总是允许本地回环地址连接（不检查 ip 数据库封禁）
}

func (s *SshRuleListConfig) setDefault() {
	for _, r := range s.RuleList {
		r.setDefault()
	}

	s.DefaultBanned.SetDefaultEnable()
	s.AlwaysAllowIntranet.SetDefaultDisable()
	s.AlwaysAllowLoopback.SetDefaultEnable()

	return
}

func (s *SshRuleListConfig) check() (err ConfigError) {
	if !s.DefaultBanned.IsEnable(false) {
		_ = NewConfigWarning("ssh recommends setting the default policy to banned")
	}

	for _, r := range s.RuleList {
		err := r.check()
		if err != nil && err.IsError() {
			return err
		}
	}

	return nil
}
