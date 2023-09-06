package engine

import (
	"fireman/internal/command"
	"fireman/internal/database"
	"fireman/internal/rules"
	"fireman/internal/util"
	"fmt"
	"github.com/c-bata/go-prompt"
	"gorm.io/gorm"
	"log"
	"os"
	"strings"
)

type HF func(string) error

type engine struct {
	db       *gorm.DB
	recorder *recorder //记录器
	output   *output   //输出器
	handMap  map[string]HF
	op       *Option
}
type recorder struct {
	//记录器，用于记录运行时信息
	barrel      []*database.Resource //用于存放资源
	workBarrel  []*database.Resource //用于存放用户选择的资源
	terminalMap map[string]command.Performer
	wtc         *rules.WaterCannon
	MessagePipe chan message
}

type Option struct {
	attrMap map[string]string
}

func (e *engine) hand(s string) {
	parts := strings.Fields(s)
	if len(parts) < 2 {
		fmt.Println("Invalid command format")
		return
	}

	//用于记录命令执行情况到文件中
	go e.output.Show(e.recorder.MessagePipe)

	cmd := parts[0]
	args := parts[1:]
	if _, ok := e.handMap[cmd]; !ok {
		fmt.Println("未知命令：", cmd)
		return
	}
	err := e.handMap[cmd](strings.Join(args, " "))
	if err != nil {
		fmt.Printf("执行%s模块发生错误：%s\n", cmd, err.Error())
	}

}

func (e *engine) useHand(s string) error {
	var err error
	e.recorder.workBarrel = e.recorder.workBarrel[:0]
	par := strings.Split(s, ",")
	if strings.Join(par, "") == "?" {
		for i, r := range e.recorder.barrel {
			fmt.Printf("ID:%d\t%s", i+1, r)
		}
		return nil
	}
	for _, p := range par {
		i := util.Atoi(p) - 1
		if i < 0 {
			continue
		}
		e.recorder.workBarrel = append(e.recorder.workBarrel, e.recorder.barrel[i])
	}

	if strings.ToUpper(s) == "ALL" {
		e.recorder.workBarrel = append(e.recorder.workBarrel, e.recorder.barrel...)
	}

	//初始化选择的客户端，如果已经存在判断其是否存活
	for _, r := range e.recorder.workBarrel {
		key := fmt.Sprintf("%s:%d:%s", r.IP, r.Port, r.User)
		if _, ok := e.recorder.terminalMap[key]; ok {
			//终端键值存在，判断其是否存活,不存活重新初始化
			if e.recorder.terminalMap[key] != nil {
				if e.recorder.terminalMap[key].Ping() {
					continue
				}
			}
		}
		e.recorder.terminalMap[key], err = command.GetPerformer(r)
		if err != nil {
			fmt.Printf("初始化:%s发送错误:%s\n", key, err)
		}
	}
	return nil
}

func (e *engine) cmdHand(s string) error {
	var seed rules.Seed

	if len(e.recorder.workBarrel) == 0 {
		return fmt.Errorf("请选择需要执行命令的资源！")
	}

	seed = e.recorder.wtc.Stuffing(s)
	if seed == nil {
		seed = rules.DefaultSeed
	}
	for _, r := range e.recorder.workBarrel {
		key := fmt.Sprintf("%s:%d:%s", r.IP, r.Port, r.User)
		if e.recorder.terminalMap[key] == nil {
			continue
		}
		date, _ := seed(e.recorder.terminalMap[key], s)
		//date = []byte(util.Rstrip(string(date), "^^"))
		mes := message{name: key, date: date, cmd: s}
		fmt.Printf("%s(%s):\n%s\n", mes.name, mes.cmd, mes.date)
		e.recorder.MessagePipe <- mes

	}
	return nil
}
func (e *engine) webShellScan(target string) error {
	return e.fileScan("./static/webshellscan_linux.zip", target)
}

func (e *engine) fileScan(source string, target string) error {
	isup := false
	if len(e.recorder.workBarrel) == 0 {
		return fmt.Errorf("请选择需要执行命令的资源！")
	}
	for _, r := range e.recorder.workBarrel {
		key := fmt.Sprintf("%s:%d:%s", r.IP, r.Port, r.User)
		if e.recorder.terminalMap[key] == nil {
			continue
		}
	scan:
		date, err := rules.ScanSeed(e.recorder.terminalMap[key], source, target, isup)
		if err != nil {
			if strings.Contains(err.Error(), "上传") {
				fmt.Printf("上传文件到%s服务器报错：%s\n", key, date)
				var s string
				fmt.Printf("请手动上传%s文件到%s服务器的/tmp目录，并且解压赋予yara执行权限\n", key, source)
				fmt.Printf("是否完成(Y/N)：")
				fmt.Scanln(&s)
				if strings.ToUpper(s) == "Y" {
					isup = true
					goto scan
				} else {
					fmt.Printf("跳过执行当前资源：%s\n", key)
					continue
				}
			}
		}
		mes := message{name: key, date: date, cmd: "fileScan"}
		fmt.Printf("%s(%s):\n%s\n", mes.name, mes.cmd, mes.date)
		e.recorder.MessagePipe <- mes
	}
	return nil
}

func (e *engine) uploadHand(s string) error {
	if len(e.recorder.workBarrel) == 0 {
		return fmt.Errorf("请选择需要执行命令的资源！")
	}
	if len(e.recorder.workBarrel) > 1 {
		return fmt.Errorf("当前选择资源数量大于一，上传文件请选择一个资源！")
	}
	r := e.recorder.workBarrel[0]
	key := fmt.Sprintf("%s:%d:%s", r.IP, r.Port, r.User)
	if e.recorder.terminalMap[key] == nil {
		return fmt.Errorf("当前选择资源无初始化！")
	}
	l := strings.Fields(s)
	if len(l) < 2 || len(l) > 2 {
		return fmt.Errorf("命令格式错误，示例：upload srcFile RemotePath")
	}
	srcFile, remotePath := l[0], l[1]
	i, err := e.recorder.terminalMap[key].UploadFile(srcFile, remotePath)
	if err == nil {
		fmt.Printf("成功上传文件：%s\t上传大小：%d\n", srcFile, i)
	}
	return err
}

func (e *engine) downloadHand(s string) error {
	if len(e.recorder.workBarrel) == 0 {
		return fmt.Errorf("请选择需要执行命令的资源！")
	}
	if len(e.recorder.workBarrel) > 1 {
		return fmt.Errorf("当前选择资源数量大于一，上传文件请选择一个资源！")
	}
	r := e.recorder.workBarrel[0]
	key := fmt.Sprintf("%s:%d:%s", r.IP, r.Port, r.User)
	if e.recorder.terminalMap[key] == nil {
		return fmt.Errorf("当前选择资源无初始化！")
	}
	l := strings.Fields(s)
	if len(l) < 2 || len(l) > 2 {
		return fmt.Errorf("命令格式错误，示例：download remoteFile LocalPath")
	}
	srcFile, remotePath := l[0], l[1]
	i, err := e.recorder.terminalMap[key].DownloadFile(srcFile, remotePath)
	if err == nil {
		fmt.Printf("成功下载文件：%s\t上传大小：%d\n", srcFile, i)
	}
	return err
}

func (e *engine) setHand(s string) error {
	var key string
	var value string
	if s == "?" {
		for k, v := range e.op.attrMap {
			fmt.Printf("%s:%v\n", k, v)
		}
		return nil
	}
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return fmt.Errorf("参数设置需要键和值，示例：set  OutPath ./static/123.csv")
	}
	if len(parts) > 2 {
		key, value = parts[0], strings.Join(parts[1:], " ")
	} else {
		key, value = parts[0], parts[1]
	}
	if _, ok := e.op.attrMap[key]; !ok {
		return fmt.Errorf("当前设置的属性名{%s}不存在", key)
	}
	e.op.attrMap[key] = strings.ReplaceAll(value, "\"", "")
	return nil
}

func (e *engine) checkAll(s string) error {
	fmt.Printf("开始检查所有检查项，日志输出路径为./static/all.csv\n")
	parmap := util.SToParMap(s)
	if _, ok := parmap["path"]; !ok {
		return fmt.Errorf("请确定检查路径，示例：checkALL time=7 path=/var/www")
	}
	if _, ok := parmap["time"]; !ok {
		return fmt.Errorf("请确定检查时间，示例：checkALL time=7 path=/var/www")
	} else {
		if util.Atoi(parmap["time"]) == -1 {
			return fmt.Errorf("请输入正确的时间，单位为天，示例：checkALL time=7 path=/var/www")
		}
	}
	course := map[string]string{
		"可登陆用户":              "User",
		"密码为空的用户":            "User empty",
		"定时任务":               "Cron",
		"敏感历史命令":             "History",
		"网络监听和连接情况":          "Network",
		"SSH登录失败的IP地址":       "SSH FIP",
		"SSH登录失败的用户":         "SSH FUSER",
		"SSH登录成功的IP地址":       "SSH SIP",
		"SSH登录成功的详细信息":       "SSH SINFO",
		"查看[time]天内被修改的定时任务": "Find Cron [time]",
		"查看[time]天内被修改的启动项":  "Find StartUp [time]",
		"查看[time]天内被修改的进程项":  "Find OS [time]",
		"查看[time]天内被修改的所有文件": "Find Time [path] [time]",
	}
	//选择所有资源
	e.useHand("ALL")
	e.op.attrMap["OutPath"] = "./static/all.csv"
	for k, v := range course {
		k = strings.ReplaceAll(k, "[time]", parmap["time"])
		v = strings.ReplaceAll(v, "[time]", parmap["time"])
		v = strings.ReplaceAll(v, "[path]", parmap["path"])
		fmt.Printf("正在检查：%s\n", k)
		e.cmdHand(v)
	}
	e.op.attrMap["OutPath"] = ""
	return nil
}

func NewEngine(db *gorm.DB) *engine {
	e := &engine{
		db: db,
		recorder: &recorder{
			barrel:      make([]*database.Resource, 1),
			workBarrel:  make([]*database.Resource, 0),
			terminalMap: make(map[string]command.Performer),
			MessagePipe: make(chan message, 1),
		},
		output:  &output{},
		handMap: make(map[string]HF),
		op: &Option{attrMap: map[string]string{
			"OutPath": "",
		}},
	}
	e.init()
	return e
}

func (e *engine) init() {
	var err error
	e.handMap["use"] = e.useHand
	e.handMap["cmd"] = e.cmdHand
	e.handMap["set"] = e.setHand
	e.handMap["exit"] = e.close
	e.handMap["upload"] = e.uploadHand
	e.handMap["download"] = e.downloadHand
	e.handMap["checkAll"] = e.checkAll
	e.handMap["webShellScan"] = e.webShellScan
	e.recorder.barrel, err = database.GetResourceAll(e.db)
	util.PrintErr(err)

	e.recorder.wtc = rules.ReloadWater()
	e.output.op = e.op
}

func (e *engine) Run() {
	// 创建一个 bufio.Scanner 以从标准输入中获取用户输入
	title := ` 
███████╗██╗██╗     ███████╗███╗   ███╗ █████╗ ███╗   ██╗
██╔════╝██║██║     ██╔════╝████╗ ████║██╔══██╗████╗  ██║
█████╗  ██║██║     █████╗  ██╔████╔██║███████║██╔██╗ ██║
██╔══╝  ██║██║     ██╔══╝  ██║╚██╔╝██║██╔══██║██║╚██╗██║
██║     ██║███████╗███████╗██║ ╚═╝ ██║██║  ██║██║ ╚████║
╚═╝     ╚═╝╚══════╝╚══════╝╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝
		作者：xiaoliu	版本：1.0
`
	fmt.Printf("%s\n", title)

	// 创建一个新的提示器
	p := prompt.New(
		e.hand,
		completer,
		prompt.OptionTitle("fireman"),
		prompt.OptionPrefix(">>> "),
		prompt.OptionInputTextColor(prompt.Blue),
		prompt.OptionCompletionOnDown(),
	)

	// 运行提示器
	p.Run()

}

func (e *engine) close(string) error {
	//关闭所有的客户端连接
	log.Println("正在关闭所有客户端连接...")
	for _, c := range e.recorder.terminalMap {
		if c != nil {
			c.Close()
		}
	}
	os.Exit(0)
	return nil
}
