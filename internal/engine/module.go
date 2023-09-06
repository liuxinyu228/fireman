package engine

import (
	"fireman/internal/util"
	"fmt"
	"github.com/c-bata/go-prompt"
	"log"
)

type message struct {
	name string
	date []byte
	cmd  string
}

type output struct {
	op *Option
}

func (o *output) Show(pip chan message) {
	for {
		//用于输出到文件操作
		mes := <-pip
		if o.op.attrMap["OutPath"] == "" {
			continue
		}
		err := util.CreateOutFile(o.op.attrMap["OutPath"], "csv")
		if err != nil {
			log.Printf("创建文件时发生错误：%s", err)
		}
		err = util.AppendFile(o.op.attrMap["OutPath"], fmt.Sprintf("%s,\"%s\",\"%s\"\n", mes.name, mes.cmd, util.Escape(string(mes.date))))
		if err != nil {
			log.Printf("写入文件时发生错误：%s", err)
		}
	}
}

// pormpt 输出提示器
// 提示自动完成的建议
func completer(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "use", Description: "选择需要执行命令的资源，示例：use ID1,ID2;"},
		{Text: "use all", Description: "选择资源库中的所有资源"},
		{Text: "use ?", Description: "显示数据库中的全部资源"},

		{Text: "cmd", Description: "需要执行的命令，示例：cmd ip a"},
		{Text: "cmd [CMD] sudo", Description: "在已有命令模块后跟上sudo，尝试提权执行命令"},
		{Text: "cmd User", Description: "查询可登陆系统的用户"},
		{Text: "cmd User empty", Description: "查询密码为空的用户"},
		{Text: "cmd Cron", Description: "检查所有用户的定时任务"},
		{Text: "cmd History", Description: "检查所有用户的敏感历史命令"},
		{Text: "cmd Network", Description: "检查机器网络监听和连接情况"},
		{Text: "cmd Pid [PID]", Description: "根据pid获取进程路径等信息"},
		{Text: "cmd SSH FIP", Description: "查看SSH登录失败的IP地址"},
		{Text: "cmd SSH FUSER", Description: "查看SSH登录失败的用户名称"},
		{Text: "cmd SSH SIP", Description: "查看SSH登录成功的IP地址"},
		{Text: "cmd SSH SINFO", Description: "查看SSH登录成功的日期、用户名、IP"},
		{Text: "cmd Find Cron [day]", Description: "查看系统各个级别定时任务目录中，n天内被修改的文件（参数为天数）"},
		{Text: "cmd Find StartUp [day]", Description: "查看系统启动项目录中，n天内被修改的文件（参数为天数）"},
		{Text: "cmd Find OS [day]", Description: "查看系统重要目录中，n天内被修改的文件（参数为天数）"},
		{Text: "cmd Find Time [path [day [postfix]]]", Description: "查看系统中指定时间内的文件的修改"},
		{Text: "cmd Find perm [path [perm [postfix]]]", Description: "查看系统中指定时间内存在修改的具有特定权限的文件"},
		{Text: "checkAll time=[] path=[]", Description: "执行所有检查项，time用于指定时间的项，path用于需要指定路径的项"},

		{Text: "upload srcFile remotePath", Description: "用于上传文件到远程服务器"},
		{Text: "download remoteFile localPath", Description: "用于下载文件到本地服务器"},

		{Text: "webShellScan [targetDir]", Description: "扫描指定目录下可能存在的WebShell"},

		{Text: "set", Description: "设置程序运行时的参数"},
		{Text: "set OutPath [path]", Description: "设置命令回显输出文件路径"},
		{Text: "set ?", Description: "显示当前配置详细"},

		{Text: "exit 0", Description: "退出程序"},
	}

	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}
