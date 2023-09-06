package rules

import (
	"fireman/internal/command"
	"fireman/internal/util"
	"fmt"
	"os"
	"strings"
)

type Seed func(performer command.Performer, cmd string) ([]byte, error)

type WaterCannon struct {
	mag map[string]Seed
}

func ReloadWater() *WaterCannon {
	w := &WaterCannon{
		mag: make(map[string]Seed),
	}
	w.mag["User"] = UserSeed
	w.mag["Cron"] = crondSeed
	w.mag["History"] = historySeed
	w.mag["Network"] = networkSeed
	w.mag["Pid"] = pidSeed
	w.mag["SSH"] = sshSeed
	w.mag["Find"] = findSeed
	return w
}

func (w *WaterCannon) Stuffing(s string) Seed {
	var cmd string

	cmd = s
	if len(strings.Fields(s)) >= 2 {
		cmd = strings.Fields(s)[0]
	}

	if _, ok := w.mag[cmd]; ok {
		return w.mag[cmd]
	}
	return nil
}

func DefaultSeed(performer command.Performer, cmd string) ([]byte, error) {
	hand := performer.Actuator
	if strings.Contains(cmd, "sudo") {
		hand = performer.SudoActuator
	}
	rule := strings.Split(cmd, "sudo")[0]
	msg, err := hand(rule)
	if err != nil {
		msg = performer.GetError()
	}
	return msg, err
}

func UserSeed(performer command.Performer, cmd string) ([]byte, error) {
	//查看密码为空且可以登陆系统的用户
	var rules string
	var msg []byte
	var err error
	if strings.Contains(cmd, "empty") {
		rules = "awk -F: 'length($2)==0 {print $1}' /etc/shadow"
		msg, err = performer.SudoActuator(rules)
		if err != nil {
			msg = performer.GetError()
		}
		return msg, err
	}
	rules = "grep '/bin/bash' /etc/passwd | cut -d: -f1 | tr '\\n' '^'"
	msg, err = performer.Actuator(rules)
	if err != nil {
		msg = performer.GetError()
		return msg, err
	}
	return msg, err
}

func crondSeed(performer command.Performer, cmd string) ([]byte, error) {
	//查看所有用户的定时任务
	ruleMap := map[string]string{
		"CENTOS": "ls /var/spool/cron | xargs -I {} cat /var/spool/cron/{}",
		"REDHAT": "ls /var/spool/cron | xargs -I {} cat /var/spool/cron/{}",
		"UBUNTU": "ls /var/spool/cron/crontabs | xargs -I {} cat /var/spool/cron/crontabs/{}",
	}
	return osMapHand(performer, ruleMap)
}

func historySeed(performer command.Performer, cmd string) ([]byte, error) {
	//查看所有用户的敏感历史命令
	var result []byte
	var msg []byte
	var err error
	rule := "cat /home/{}/.bash_history |grep -E \\\"wget|curl|http|rsync|sftp|ssh|scp|rcp|python|java|chmod|ftp|bash｜zip|tar\\\""
	date, err := UserSeed(performer, cmd)
	if err != nil {
		return date, err
	}
	users := strings.Split(string(date), "^")
	for _, u := range users {
		if u == "root" || u == "" {
			continue
		}
		r := strings.ReplaceAll(rule, "{}", u)
		msg, err = performer.SudoActuator(r)
		if err != nil {
			msg = performer.GetError()
		}
		result = append(result, msg...)
	}
	rule = "cat /root/.bash_history |grep -E \\\"wget|curl|http|rsync|sftp|ssh|scp|rcp|python|java|chmod|ftp|bash｜zip|tar\\\""
	msg, err = performer.SudoActuator(rule)
	if err != nil {
		msg = performer.GetError()
	}
	result = append(result, msg...)
	return result, err
}

func networkSeed(performer command.Performer, cmd string) ([]byte, error) {
	rule := "netstat -tanp"
	msg, err := performer.SudoActuator(rule)
	if err != nil {
		msg = performer.GetError()
	}
	if !strings.Contains(string(msg), "state") {
		rule = "ss -tanp"
	}
	msg, err = performer.SudoActuator(rule)
	if err != nil {
		msg = performer.GetError()
	}
	return msg, err
}

func pidSeed(performer command.Performer, cmd string) ([]byte, error) {
	pid := strings.Fields(cmd)[1:]
	rule := fmt.Sprintf("ls -all /proc/%s |grep \"exe\"", strings.Join(pid, ""))
	msg, err := performer.SudoActuator(rule)
	if err != nil {
		msg = performer.GetError()
	}
	return msg, err
}

func sshSeed(performer command.Performer, cmd string) ([]byte, error) {
	var hl []string
	var result []byte
	Mmap := map[string]Seed{
		"FIP":   sshFIPSeed,
		"FUSER": sshFUserSeed,
		"SIP":   sshSIPSeed,
		"SINFO": sshSInfoSeed,
	}
	cmd = strings.ToUpper(cmd)
	l := strings.Fields(cmd)
	if len(l) >= 2 {
		hl = append(hl, l[1:]...)
	} else {
		hl = append(hl, "FIP")
		hl = append(hl, "FUSER")
		hl = append(hl, "SIP")
		hl = append(hl, "SINFO")
	}

	for _, v := range hl {
		if _, ok := Mmap[v]; !ok {
			fmt.Printf("%s子模块名称未知\n", v)
			continue
		}
		msg, err := Mmap[v](performer, cmd)
		if err != nil {
			fmt.Printf("%s子模块执行时发生错误:%s\n", v, msg)
			continue
		}
		result = append(result, []byte(fmt.Sprintf("%s:\t", v))...)
		result = append(result, msg...)
	}
	return result, nil
}

func sshFIPSeed(performer command.Performer, cmd string) ([]byte, error) {
	// 定位有哪些IP在爆破
	ruleMap := map[string]string{
		"DEBIAN":   "grep \"Failed\" /var/log/auth.log*|grep -E -o \\\"(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\\"|uniq -c",
		"UBUNTU":   "grep \"Failed\" /var/log/auth.log*|grep -E -o \\\"(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\\"|uniq -c",
		"REDHAT":   "grep \"Failed\" /var/log/secure*|grep -E -o \\\"(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\\"|uniq -c",
		"CENTOS":   "grep \"Failed\" /var/log/secure*|grep -E -o \\\"(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\\"|uniq -c",
		"OPENSUSE": "grep \"Failed\" /var/log/messages*|grep -E -o \\\"(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\\"|uniq -c",
	}
	return osMapHand(performer, ruleMap)
}

func sshFUserSeed(performer command.Performer, cmd string) ([]byte, error) {
	//被爆破用户名是什么
	ruleMap := map[string]string{
		"CENTOS":   "grep 'Failed' /var/log/secure*| perl -e 'while(<>){ /for(.*?) from/; print \\\"\\$1\\\\n\\\";}' | sort | uniq -c | sort -nr",
		"REDHAT":   "grep 'Failed' /var/log/secure*| perl -e 'while(<>){ /for(.*?) from/; print \\\"\\$1\\\\n\\\";}' | sort | uniq -c | sort -nr",
		"UBUNTU":   "grep 'Failed' /var/log/auth.log* | perl -e 'while(<>){ /for(.*?) from/; print \\\"\\$1\\\\n\\\";}' | sort | uniq -c | sort -nr",
		"DEBIAN":   "grep 'Failed' /var/log/auth.log* | perl -e 'while(<>){ /for(.*?) from/; print \\\"\\$1\\\\n\\\";}' | sort | uniq -c | sort -nr",
		"OPENSUSE": "grep 'Failed' /var/log/messages*| perl -e 'while(<>){ /for(.*?) from/; print \\\"\\$1\\\\n\\\";}' | sort | uniq -c | sort -nr",
	}
	return osMapHand(performer, ruleMap)
}

func sshSIPSeed(performer command.Performer, cmd string) ([]byte, error) {
	//查看ssh登录成功的ip和次数
	ruleMap := map[string]string{
		"DEBIAN":   "grep \"Accepted\" /var/log/auth.log*|grep -E -o \\\"(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\\"| sort | uniq -c | sort -nr",
		"UBUNTU":   "grep \"Accepted\" /var/log/auth.log*|grep -E -o \\\"(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\\"| sort | uniq -c | sort -nr",
		"REDHAT":   "grep \"Accepted\" /var/log/secure*|grep -E -o \\\"(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\\"| sort | uniq -c | sort -nr",
		"CENTOS":   "grep \"Accepted\" /var/log/secure*|grep -E -o \\\"(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\\"| sort | uniq -c | sort -nr",
		"OPENSUSE": "grep \"Accepted\" /var/log/messages*|grep -E -o \\\"(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\\"| sort | uniq -c | sort -nr",
	}
	return osMapHand(performer, ruleMap)
}

func sshSInfoSeed(performer command.Performer, cmd string) ([]byte, error) {
	//ssh登录成功的用户详情
	ruleMap := map[string]string{
		"CENTOS":   "grep \\\"Accepted\\\" /var/log/secure* | awk '{print \\$1,\\$2,\\$3,\\$9,\\$11}'",
		"REDHAT":   "grep \\\"Accepted\\\" /var/log/secure* | awk '{print \\$1,\\$2,\\$3,\\$9,\\$11}'",
		"UBUNTU":   "grep \\\"Accepted\\\" /var/log/auth.log* | awk '{print \\$1,\\$2,\\$3,\\$9,\\$11}'",
		"DEBIAN":   "grep \\\"Accepted\\\" /var/log/auth.log* | awk '{print \\$1,\\$2,\\$3,\\$9,\\$11}'",
		"OPENSUSE": "grep \\\"Accepted\\\" /var/log/messages* | awk '{print \\$1,\\$2,\\$3,\\$9,\\$11}'",
	}
	return osMapHand(performer, ruleMap)
}

func findSeed(performer command.Performer, cmd string) ([]byte, error) {
	var result []byte
	Mmap := map[string]Seed{
		"Cron":    findCronSeed,
		"StartUp": findStartupSeed,
		"OS":      findOSSeed,
		"Time":    findChangeForDaySeed,
		"Perm":    findChangeForPermSeed,
	}
	l := strings.Fields(cmd)
	if _, ok := Mmap[l[1]]; !ok {
		return nil, fmt.Errorf("%s子模块名称未知\n", l[1])
	}
	result = append(result, []byte(fmt.Sprintf("%s:\t", l[1]))...)
	msg, err := Mmap[l[1]](performer, cmd)
	result = append(result, msg...)
	return msg, err
}

func findCronSeed(performer command.Performer, cmd string) ([]byte, error) {
	//查看系统各个级别定时任务目录中，n天内被修改的文件（参数为天数）
	p := util.ParamToA(strings.Fields(cmd)[2])
	if p == -1 {
		return nil, fmt.Errorf("错误的时间参数：%s, 示例：cmd Find cron 7", strings.Fields(cmd)[2])
	}
	rules := []string{
		fmt.Sprintf("find /etc/cron.d/ -mtime -%d", p),
		fmt.Sprintf("find /etc/cron.hourly/ -mtime -%d", p),
		fmt.Sprintf("find /etc/cron.daily/ -mtime -%d", p),
		fmt.Sprintf("find /etc/cron.weekly/ -mtime -%d", p),
		fmt.Sprintf("find /etc/cron.monthly/ -mtime -%d", p),
	}
	return ruleListHand(performer, rules)
}

func findStartupSeed(performer command.Performer, cmd string) ([]byte, error) {
	//查看系统启动项目录中，n天内被修改的文件（参数为天数）
	p := util.ParamToA(strings.Fields(cmd)[2])
	if p == -1 {
		return nil, fmt.Errorf("错误的时间参数：%s, 示例：cmd Find startup 7", strings.Fields(cmd)[2])
	}
	rules := []string{
		fmt.Sprintf("cat /etc/rc.local"),
		fmt.Sprintf("find /etc/rc0.d/ -mtime -%d", p),
		fmt.Sprintf("find /etc/rc1.d/ -mtime -%d", p),
		fmt.Sprintf("find /etc/rc2.d/ -mtime -%d", p),
		fmt.Sprintf("find /etc/rc3.d/ -mtime -%d", p),
		fmt.Sprintf("find /etc/rc4.d/ -mtime -%d", p),
		fmt.Sprintf("find /etc/rc5.d/ -mtime -%d", p),
		fmt.Sprintf("find /etc/rc6.d/ -mtime -%d", p),
		fmt.Sprintf("find /etc/init/rc.d/ -mtime -%d", p),
	}
	return ruleListHand(performer, rules)
}

func findOSSeed(performer command.Performer, cmd string) ([]byte, error) {
	p := util.ParamToA(strings.Fields(cmd)[2])
	if p == -1 {
		return nil, fmt.Errorf("错误的时间参数：%s, 示例：cmd Find os 7", strings.Fields(cmd)[2])
	}
	//查看系统进程是否被劫持
	rule := fmt.Sprintf("find /usr/bin/ /usr/sbin/ /bin/ /usr/local/bin/ -mtime -%d", p)
	msg, err := performer.SudoActuator(rule)
	if err != nil {
		msg = performer.GetError()
	}
	return msg, err
}

func findChangeForDaySeed(performer command.Performer, cmd string) ([]byte, error) {
	//根据时间查询指定目录修改的文件
	l := strings.Fields(cmd)
	if len(l) < 4 {
		return nil, fmt.Errorf("命令格式错误，示例：cmd find Time /var 7 [php]")
	}
	rule := fmt.Sprintf("find %s -mtime -%s", l[2], l[3])
	if len(l) == 5 {
		rule = fmt.Sprintf("find %s -mtime -%s -name \"*.%s\"", l[2], l[3], l[4])
	}
	msg, err := performer.SudoActuator(rule)
	if err != nil {
		msg = performer.GetError()
	}
	return msg, nil
}

func findChangeForPermSeed(performer command.Performer, cmd string) ([]byte, error) {
	//根据时间查询指定目录修改的文件
	l := strings.Fields(cmd)
	if len(l) < 4 {
		return nil, fmt.Errorf("命令格式错误，示例：cmd find Perm /var 777 [php]")
	}
	rule := fmt.Sprintf("find %s -perm %s", l[2], l[3])
	if len(l) == 5 {
		rule = fmt.Sprintf("find %s -perm -%s -name \"*.%s\"", l[2], l[3], l[4])
	}
	msg, err := performer.SudoActuator(rule)
	if err != nil {
		msg = performer.GetError()
	}
	return msg, nil
}

func osMapHand(performer command.Performer, ruleMap map[string]string) ([]byte, error) {
	//针对不同发行版本命令不同且不需要参数的函数的复用函数
	if _, ok := ruleMap[performer.OSName()]; !ok {
		return nil, fmt.Errorf("%s 类型主机未知", performer.OSName())
	}
	rule := ruleMap[performer.OSName()]
	msg, err := performer.SudoActuator(rule)
	if err != nil {
		msg = []byte(err.Error())
	}
	return msg, err
}

func ruleListHand(performer command.Performer, rules []string) ([]byte, error) {
	var result []byte
	for _, r := range rules {
		fmt.Println(r)
		msg, err := performer.SudoActuator(r)
		if err != nil {
			msg = []byte(err.Error())
		}
		if util.Strip(string(msg)) == "" {
			continue
		}
		result = append(result, msg...)
	}
	return result, nil
}

func ScanSeed(performer command.Performer, sourceFile string, targetPath string, isup bool) ([]byte, error) {
	var cmd string
	if !isup {
		fmt.Printf("开始上传yara扫描引擎...\n")
		_, err := performer.UploadFile(sourceFile, "/tmp/")
		if err != nil {
			return nil, fmt.Errorf("上传扫描引擎报错：%s", err)
		}
		sinfo, _ := os.Stat(sourceFile)
		cmd = fmt.Sprintf("unzip -o /tmp/%s && chmod 744 /tmp/yara", sinfo.Name())
		_, err = performer.Actuator(cmd)
		if err != nil {
			return performer.GetError(), fmt.Errorf("解压上传文件失败：%s", err)
		}
	}
	cmd = fmt.Sprintf("/tmp/yara /tmp/rules.yar -r %s", targetPath)
	fmt.Printf("正在扫描...\t%s\n", cmd)
	msg, err := performer.SudoActuator(cmd)
	if err != nil {
		msg = performer.GetError()
	}
	return msg, err
}
