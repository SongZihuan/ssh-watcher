package sshserver

import (
	"fmt"
	"github.com/SongZihuan/ssh-watcher/src/api/apiip"
	"github.com/SongZihuan/ssh-watcher/src/config"
	"github.com/SongZihuan/ssh-watcher/src/database"
	"github.com/SongZihuan/ssh-watcher/src/ipcheck"
	"github.com/SongZihuan/ssh-watcher/src/logger"
	"github.com/SongZihuan/ssh-watcher/src/notify"
	"github.com/SongZihuan/ssh-watcher/src/redisserver"
	"github.com/pires/go-proxyproto"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type SshServer struct {
	status atomic.Int32
	config *config.SshForwardConfig

	ln4              net.Listener
	ln4Proxy         bool
	ln4Cross         bool
	ln4Target        *net.TCPAddr
	ln4TargetNetwork string

	ln6              net.Listener
	ln6Proxy         bool
	ln6Cross         bool
	ln6Target        *net.TCPAddr
	ln6TargetNetwork string

	swg      sync.WaitGroup
	allconn  sync.Map
	stopchan chan bool
}

func NewSshServer(cfg *config.SshForwardConfig) (*SshServer, error) {
	if cfg.ResolveIPv4DestAddress == nil && cfg.ResolveIPv6DestAddress == nil {
		return nil, fmt.Errorf("no dest address")
	}

	res := &SshServer{
		config: cfg,
	}

	res.status.Store(StatusReady)

	return res, nil
}

func (s *SshServer) Start() (err error) {
	if s.ln4 != nil || s.status.Load() != StatusReady {
		return nil
	}

	if ipcheck.SupportIPv4() {
		if s.config.ResolveIPv4DestAddress != nil {
			_ln4, err := net.ListenTCP("tcp4", s.config.ResolveIPv4SrcAddress)
			if err != nil {
				return fmt.Errorf("listen %d on tcp4 failed: %s", s.config.SrcPort, err.Error())
			}

			if s.config.IPv4SrcServerProxy.IsEnable(false) {
				s.ln4 = &proxyproto.Listener{
					Listener: _ln4,
				}
				s.ln4Proxy = true
				s.ln4Cross = false
				s.ln4Target = s.config.ResolveIPv4DestAddress
				s.ln4TargetNetwork = "tcp4"
			} else {
				s.ln4 = _ln4
				s.ln4Proxy = false
				s.ln4Cross = false
				s.ln4Target = s.config.ResolveIPv4DestAddress
				s.ln4TargetNetwork = "tcp4"
			}
		} else if s.config.Cross && s.config.ResolveIPv6DestAddress != nil {
			_ln4, err := net.ListenTCP("tcp4", s.config.ResolveIPv4SrcAddress)
			if err != nil {
				return fmt.Errorf("listen %d on tcp4 failed: %s", s.config.SrcPort, err.Error())
			}

			s.ln4 = _ln4
			s.ln4Proxy = false
			s.ln4Cross = true
			s.ln4Target = s.config.ResolveIPv6DestAddress
			s.ln4TargetNetwork = "tcp6"
		}
	} else {
		s.ln4 = nil
		s.ln4Proxy = false
		s.ln4Cross = false
		s.ln4Target = nil
		s.ln4TargetNetwork = ""
	}

	if ipcheck.SupportIPv6() {
		if s.config.ResolveIPv6DestAddress != nil {
			_ln6, err := net.ListenTCP("tcp6", s.config.ResolveIPv6SrcAddress)
			if err != nil {
				return fmt.Errorf("listen %d on tcp6 failed: %s", s.config.SrcPort, err.Error())
			}

			if s.config.IPv6SrcServerProxy.IsEnable(false) {
				s.ln6 = &proxyproto.Listener{
					Listener: _ln6,
				}
				s.ln6Proxy = true
				s.ln6Cross = false
				s.ln6Target = s.config.ResolveIPv6DestAddress
				s.ln6TargetNetwork = "tcp6"
			} else {
				s.ln6 = _ln6
				s.ln6Proxy = false
				s.ln6Cross = false
				s.ln6Target = s.config.ResolveIPv6DestAddress
				s.ln6TargetNetwork = "tcp6"
			}
		} else if s.config.Cross && s.config.ResolveIPv4DestAddress != nil {
			_ln6, err := net.ListenTCP("tcp6", s.config.ResolveIPv6SrcAddress)
			if err != nil {
				return fmt.Errorf("listen %d on tcp6 failed: %s", s.config.SrcPort, err.Error())
			}

			s.ln6 = _ln6
			s.ln6Proxy = false
			s.ln6Cross = true
			s.ln6Target = s.config.ResolveIPv4DestAddress
			s.ln6TargetNetwork = "tcp4"
		}
	} else {
		s.ln6 = nil
		s.ln6Proxy = false
		s.ln6Cross = false
		s.ln6Target = nil
		s.ln6TargetNetwork = ""
	}

	if s.ln4 == nil && s.ln6 == nil {
		return fmt.Errorf("no listen address")
	}

	if s.ln4Target == nil && s.ln6Target == nil {
		return fmt.Errorf("no target address")
	}

	s.stopchan = make(chan bool, 4)

	if s.ln4 != nil {
		go func() {
			defer func() {
				_ = s.ln4.Close()
				s.ln4 = nil
			}()

			logger.Infof("listen on %d (ipv4) start", s.config.SrcPort)
		MainCycle:
			for {
				select {
				case <-s.stopchan:
					break MainCycle
				default:
					// pass
				}

				status := s.accept(s.ln4,
					"tcp4",
					!s.ln4Cross && s.config.IPv4DestRequestProxy.IsEnable(true),
					s.config.IPv4DestRequestProxyVersion,
					s.ln4TargetNetwork,
					s.ln4Target)
				if status == StatusStop {
					break MainCycle
				}
			}

			logger.Infof("listen on %d (ipv4) stop", s.config.SrcPort)
		}()
	}

	if s.ln6 != nil {
		go func() {
			defer func() {
				_ = s.ln6.Close()
				s.ln6 = nil
			}()

			logger.Infof("listen on %d (ipv6) start", s.config.SrcPort)
		MainCycle:
			for {
				select {
				case <-s.stopchan:
					break MainCycle
				default:
					// pass
				}

				status := s.accept(s.ln6,
					"tcp6",
					!s.ln6Cross && s.config.IPv6DestRequestProxy.IsEnable(true),
					s.config.IPv6DestRequestProxyVersion,
					s.ln6TargetNetwork,
					s.ln6Target)
				if status == StatusStop {
					break MainCycle
				}
			}

			logger.Infof("listen on %d (ipv6) stop", s.config.SrcPort)
		}()
	}

	if !s.status.CompareAndSwap(StatusReady, StatusRunning) {
		return fmt.Errorf("server run failed: can not set status")
	}

	return nil
}

func (s *SshServer) Stop() error {
	if s.ln4 == nil || !s.status.CompareAndSwap(StatusRunning, StatusStopping) {
		return nil
	}

	close(s.stopchan)

	time.Sleep(1 * time.Second)

	go func() {
		time.Sleep(time.Second * 10)

		s.allconn.Range(func(key, value any) bool {
			conn, ok := value.(net.Conn)
			if !ok {
				return true
			}

			go func() {
				_ = conn.Close()
			}()
			return true
		})
		s.allconn.Clear()
	}()

	s.swg.Wait()

	s.status.CompareAndSwap(StatusStopping, StatusFinished)
	return nil
}

func (s *SshServer) forward(remoteAddr string, conn net.Conn, target net.Conn, record *database.SshConnectRecord) {
	defer func() {
		defer func() {
			_ = recover()
		}()

		err := database.UpdateSshConnectRecord(record, "连接正常断开。")
		if err != nil {
			logger.Errorf("update ssh connect record error: %s", record)
		}
	}()

	defer func() {
		r := recover()
		if r != nil {
			if err, ok := r.(error); ok {
				logger.Panicf("ssh forward panic error: %s", err.Error())
			} else {
				logger.Panicf("ssh forward panic error: %v", r)
			}
		}
	}()

	s.swg.Add(1)
	defer s.swg.Done()

	if _, loaded := s.allconn.LoadOrStore(remoteAddr, conn); loaded {
		logger.Errorf("%s is already connected", remoteAddr)
		return
	}
	defer func() {
		s.allconn.Delete(remoteAddr)
	}()

	var stopchan1 = make(chan bool)
	var stopchan2 = make(chan bool)

	var wg sync.WaitGroup

	defer wg.Wait()

	defer func() {
		defer func() {
			_ = recover()
		}()

		_conn := conn
		conn = nil
		_ = _conn.Close()

	}()

	defer func() {
		defer func() {
			_ = recover()
		}()

		_target := target
		target = nil
		_ = _target.Close()
	}()

	go func() {
		wg.Add(1)

		defer wg.Done()

		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					logger.Panicf("failed to forward: %s", err.Error())
				} else {
					logger.Panicf("failed to forward: %v", r)
				}
			}
		}()

		defer func() {
			close(stopchan1)
		}()

		_, err := io.Copy(target, conn)
		if err != nil && conn != nil && target != nil && s.status.Load() == StatusRunning {
			logger.Errorf("failed to forward from %s to %s: %v", conn.RemoteAddr(), target.RemoteAddr(), err)
		}
	}()

	go func() {
		wg.Add(1)
		defer wg.Done()

		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					logger.Panicf("failed to forward from: %s", err.Error())
				} else {
					logger.Panicf("failed to forward from: %v", r)
				}
			}
		}()

		defer func() {
			close(stopchan2)
		}()

		_, err := io.Copy(conn, target)
		if err != nil && conn != nil && target != nil && s.status.Load() == StatusRunning {
			logger.Errorf("failed to forward from %s to %s: %v", target.RemoteAddr(), conn.RemoteAddr(), err)
		}
	}()

	select {
	case <-stopchan1:
	case <-stopchan2:
	}

	return
}

func (s *SshServer) accept(ln net.Listener, srcNetwork string, destProxy bool, destProxyVersion int, targetNetwork string, targetAddr *net.TCPAddr) string {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				logger.Panicf("listen on %d panic (error) : %s", s.config.SrcPort, err.Error())
			} else {
				logger.Panicf("listen on %d panic : %s", s.config.SrcPort, err.Error())
			}
		}
	}()

	if ln == nil {
		return StatusStop
	}

	conn, err := ln.Accept()
	if err != nil {
		logger.Errorf("listen on %d accecpt error: %s", s.config.SrcPort, err.Error())
		return StatusContinue
	}
	defer func() {
		if conn != nil {
			_ = conn.Close()
		}
	}()

	now := time.Now()

	remoteAddr := conn.RemoteAddr()
	if remoteAddr == nil {
		return StatusContinue
	}

	remoteSSHAddr, err := net.ResolveTCPAddr(srcNetwork, remoteAddr.String())
	if err != nil {
		return StatusContinue
	}

	loc, ckErr := s.remoteAddrCheck(remoteSSHAddr, targetAddr)
	if ckErr != nil {
		_, _ = s.addSshConnectRecord("", remoteSSHAddr.IP, targetAddr, loc, false, now, fmt.Sprintf("来访IP检查出现问题。%s", ckErr.Error()))
		return StatusContinue
	}

	target, err := net.DialTCP(targetNetwork, nil, targetAddr)
	if err != nil {
		logger.Errorf("Failed to connect to target %s: %v", targetAddr.String(), err)
		_, _ = s.addSshConnectRecord("", remoteSSHAddr.IP, targetAddr, loc, false, now, "无法解析来访TCP地址。")
		return StatusContinue
	}
	defer func() {
		if target != nil {
			_ = target.Close()
		}
	}()

	if destProxy {
		header := proxyproto.HeaderProxyFromAddrs(byte(destProxyVersion), remoteSSHAddr, targetAddr)
		_, err = header.WriteTo(target)
		if err != nil {
			logger.Errorf("Failed to write proxy header to target %s: %v", targetAddr.String(), err)
			_, _ = s.addSshConnectRecord("", remoteSSHAddr.IP, targetAddr, loc, false, now, "无法写入Proxy协议头部。")
			return StatusContinue
		}
	}

	record, err := s.addSshConnectRecord("", remoteSSHAddr.IP, targetAddr, loc, true, now, "允许建立连接。")
	if err != nil {
		logger.Errorf("Fail to save ssh connect record to database: %s", err.Error())
		_, _ = s.addSshConnectRecord("", remoteSSHAddr.IP, targetAddr, loc, true, now, "无法记录SSH数据，不允许建立连接。")
		return StatusContinue
	}

	_conn := conn
	_target := target
	conn = nil
	target = nil
	go s.forward(remoteAddr.String(), _conn, _target, record)

	return StatusContinue
}

func (s *SshServer) remoteAddrCheck(remoteAddr *net.TCPAddr, to *net.TCPAddr) (loc *apiip.QueryIpLocationData, err error) {
	const IspLoopback = "本地回环地址"
	const IspIntranet = "内网地址"

	ip := remoteAddr.IP
	if ip == nil {
		return nil, fmt.Errorf("无法获取IP")
	}

	isLoopback := ip.IsLoopback()
	isIntranet := isLoopback || ip.IsPrivate()

	if isLoopback && (config.GetConfig().SSH.RuleList.AlwaysAllowIntranet.IsEnable(false) || config.GetConfig().SSH.RuleList.AlwaysAllowLoopback.IsEnable(true)) {
		return &apiip.QueryIpLocationData{
			Isp: IspLoopback,
		}, nil
	}

	if !database.SshCheckIP(ip.String()) {
		return nil, fmt.Errorf("IP地址被SQLite中定义的规则（IP）封禁。")
	}

	if isIntranet && config.GetConfig().SSH.RuleList.AlwaysAllowIntranet.IsEnable(false) {
		if isLoopback {
			return &apiip.QueryIpLocationData{
				Isp: IspLoopback,
			}, nil
		}
		return &apiip.QueryIpLocationData{
			Isp: IspIntranet,
		}, nil
	}

	if isIntranet {
		loc = &apiip.QueryIpLocationData{
			Isp: "内网地址",
		}
	} else {
		var err error
		loc, err = redisserver.QueryIpLocation(ip.String())
		if err != nil {
			logger.Errorf("failed to query ip location: %s", err.Error())
			return loc, fmt.Errorf("查询IP定位失败（%s）。", err.Error())
		} else if loc == nil {
			logger.Panicf("failed to query ip location: loc is nil")
			return loc, fmt.Errorf("查询IP定位失败（loc is nil）。")
		}

		if !database.SshCheckLocationNation(loc.Nation) {
			return loc, fmt.Errorf("IP地址被SQLite中定义的规则（地区-国家）封禁。")
		}

		if !database.SshCheckLocationProvince(loc.Province) {
			return loc, fmt.Errorf("IP地址被SQLite中定义的规则（地区-省份）封禁。")
		}

		if !database.SshCheckLocationCity(loc.City) {
			return loc, fmt.Errorf("IP地址被SQLite中定义的规则（地区-城市）封禁。")
		}

		if !database.SshCheckLocationISP(loc.Isp) {
			return loc, fmt.Errorf("IP地址被SQLite中定义的规则（地区-ISP）封禁。")
		}
	}

	rcErr := s.countRulesCheck(ip, to, s.config.CountRules)
	if rcErr != nil {
		return loc, rcErr
	}

RuleCycle:
	for _, r := range config.GetConfig().SSH.RuleList.RuleList {
		if loc == nil || loc.Isp == IspIntranet {
			if r.HasLocation() {
				continue RuleCycle
			}
		} else {
			ok, err := loc.CheckLocation(&r.RuleConfig)
			if err != nil {
				logger.Errorf("check location error: %s", err.Error())
				return loc, fmt.Errorf("在配置文件规则策略中，检测IP地址错误。")
			} else if !ok {
				continue RuleCycle
			}
		}

		ok, err := r.CheckIP(ip)
		if err != nil {
			logger.Errorf("check ip error: %s", err.Error())
			return loc, fmt.Errorf("在配置文件规则策略中，检测IP信息错误。")
		} else if !ok {
			logger.Tagf("Tag 2")
			continue RuleCycle
		}

		if r.Banned.ToBool(true) { // true - 封禁
			return loc, fmt.Errorf("IP在配置文件规则策略中被封禁。")
		}

		return loc, nil
	}

	if config.GetConfig().SSH.RuleList.DefaultBanned.ToBool(true) { // true - 封禁
		return loc, fmt.Errorf("IP在配置文件默认兜底规则策略中被封禁。")
	}

	return loc, nil
}

func (s *SshServer) countRulesCheck(ip net.IP, to *net.TCPAddr, countRules []*config.SshCountRuleConfig) error {
	now := time.Now()

	if !redisserver.QuerySSHIpBanned(ip.String()) {
		return fmt.Errorf("IP在配置文件计数策略中被封禁，IP已被Redis封禁。")
	}

	if len(countRules) > 0 {
		limit := int(countRules[0].TryCount + 1) // +1防止TryCount是0
		after := now.Add(-1 * time.Second * time.Duration(countRules[0].Seconds))

		res, err := database.FindSshConnectRecord("", ip, to, limit, after)
		if err != nil {
			logger.Errorf("count rules check error: %s", err.Error())
			return fmt.Errorf("从数据库读取SSH记录异常，禁止连接。")
		}

		for _, r := range countRules {
			if s._countRulesCheck(res, r, now) {
				if r.BannedSeconds <= 0 {
					return nil // 返回是否放行，true表示放行
				}

				err := redisserver.SetSSHIpBanned(ip.String(), time.Duration(r.BannedSeconds)*time.Second)
				if err != nil {
					logger.Errorf("count rules check error: %s", err.Error())
				}
				return fmt.Errorf("IP在配置文件计数策略中被封禁, 时长 %d 秒。", r.BannedSeconds)
			}
		}
	} else {
		// 默认策略：3分钟内5次以上, 封禁10分钟
		limit := 10                              // +1防止TryCount是0
		after := now.Add(-1 * time.Second * 180) // 三分钟

		res, err := database.FindSshConnectRecord("", ip, to, limit, after)
		if err != nil {
			logger.Errorf("count rules check error: %s", err.Error())
			return fmt.Errorf("从数据库读取SSH记录异常，禁止连接。")
		}

		if len(res) > 5 {
			// 命中默认策略
			err := redisserver.SetSSHIpBanned(ip.String(), 600*time.Second)
			if err != nil {
				logger.Errorf("count rules check error: %s", err.Error())
			}
			return fmt.Errorf("IP在配置文件计数策略中被封禁, 时长 %d 秒。", 600)
		}
	}

	return nil // 没有命中封禁策略
}

func (*SshServer) _countRulesCheck(record []database.SshConnectRecord, rules *config.SshCountRuleConfig, now time.Time) bool {
	var index = 0
	after := now.Add(-1 * time.Second * time.Duration(rules.Seconds))

	for i, r := range record {
		if r.Time.After(after) {
			index = i
			break
		}
	}

	return len(record)-index > int(rules.TryCount) // 返回是否命中策略，true表示命中 (使用大于, 而不是大于等于)
}

func (s *SshServer) addSshConnectRecord(from string, fromIP net.IP, to *net.TCPAddr, loc *apiip.QueryIpLocationData, accept bool, now time.Time, mark string) (*database.SshConnectRecord, error) {
	record, err := database.AddSshConnectRecord(from, fromIP, to, accept, now, mark)
	if err != nil {
		return nil, err
	}

	if accept {
		notify.SendSshSuccess(record.From, loc, record.To, record.Mark)
	} else {
		notify.SendSshBanned(record.From, loc, record.To, record.Mark)
	}

	return record, nil
}
