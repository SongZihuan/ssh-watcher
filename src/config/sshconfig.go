package config

type SshConfig struct {
	RuleList SshRuleListConfig `yaml:",inline"`
	Forward  SshForwardConfig  `yaml:",inline"`
}

func (s *SshConfig) setDefault() {
	s.RuleList.setDefault()
	s.Forward.setDefault()
	return
}

func (s *SshConfig) check() (err ConfigError) {
	err = s.RuleList.check()
	if err != nil && err.IsError() {
		return err
	}

	err = s.Forward.check()
	if err != nil && err.IsError() {
		return err
	}

	return
}
