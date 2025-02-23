package config

type SshConfig struct {
	RuleList SshRuleListConfig `yaml:",inline"`
	Forward  SshForwardConfig  `yaml:",inline"`
}

func (t *SshConfig) setDefault() {
	t.RuleList.setDefault()
	t.Forward.setDefault()
	return
}

func (t *SshConfig) check() (err ConfigError) {
	err = t.RuleList.check()
	if err != nil && err.IsError() {
		return err
	}

	err = t.Forward.check()
	if err != nil && err.IsError() {
		return err
	}

	return
}
