# fireman批量应急响应工具

### 简介

fireman用于在维护多台服务器并且需要定时检查服务器状态的场景下。使用自带命令可一键获取相关资源信息，排查服务器是否存在可疑用户、非法外连、文件更改等高危事件。

### 使用

#### res模块

用于管理资源信息

##### 添加资源

* 编辑资源信息

  编写Source.yaml文件，可一次添加多个资源，模板如下：

  在连接资源时优先使用证书连接，如果证书解析错误再尝试使用密码连接

  ```yaml
  workspace: #资源名称
    ip: 192.168.196.143  #资源IP
    port: 22	#资源端口
    protocol: SSH #资源连接协议
    user: liuxinyu	#资源登陆用户
    passwd: #资源登陆密码
    keyfile: ./static/WorkSpace_id_rsa #资源登陆证书
    
  workspaceTwo: #资源名称
    ip: 192.168.196.144  #资源IP
    port: 22	#资源端口
    protocol: SSH #资源连接协议
    user: liuxinyu	#资源登陆用户
    passwd: qwe123!! #资源登陆密码
    keyfile:  #资源登陆证书

* 添加资源到数据

  ```cmd
  fireman.exe res add -f Source.yaml
  ```

##### 修改资源

* 修改资源密码

```
fireman.exe res update -i 192.168.196.144 -u liuxinyu -w lkjuio890!!
```

* 修改资源证书

```cmd
fireman.exe res update -i 192.168.196.144 -u liuxinyu  -k ./static/test_id_rsa
```

* 置空密码或者证书

```
fireman.exe res update -i 192.168.196.144 -u liuxinyu -w " " 
fireman.exe res update -i 192.168.196.144 -u liuxinyu -k " "
```

如果密码或者证书已经存在无法重复添加。

##### 查询资源

```
fireman.exe res query -i 192.168.196.144 -u liuxinyu

#-i或者-u条件可以单独存在
#ID:2    Name:workspace  IP:192.168.196.143      Port:22 Protocol:SSH    User:liuxinyu   Passwd:liuliu228@       KeyFile: 
```

##### 删除资源

```
fireman.exe res del -d 资源ID值
```

#### run模块

命令执行模块

```
(base) PS E:\MyGoWorkSpace\fireman> .\fireman.exe run
 
███████╗██╗██╗     ███████╗███╗   ███╗ █████╗ ███╗   ██╗
██╔════╝██║██║     ██╔════╝████╗ ████║██╔══██╗████╗  ██║
█████╗  ██║██║     █████╗  ██╔████╔██║███████║██╔██╗ ██║
██╔══╝  ██║██║     ██╔══╝  ██║╚██╔╝██║██╔══██║██║╚██╗██║
██║     ██║███████╗███████╗██║ ╚═╝ ██║██║  ██║██║ ╚████║
╚═╝     ╚═╝╚══════╝╚══════╝╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝
                作者：xiaoliu   版本：1.0

>>>
     use                                    选择需要执行命令的资源，示例：use ID1,ID2;                       
     use all                                选择资源库中的所有资源                                           
     use ?                                  显示数据库中的全部资源                                           
     cmd                                    需要执行的命令，示例：cmd ip a                                   
     cmd [CMD] sudo                         在已有命令模块后跟上sudo，尝试提权执行命令                       
     cmd User                               查询可登陆系统的用户                                             

```

交互式的命令执行，可单项排查，也可以一键排查所有项，相关命令如下：

```go
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
```

需要将命令信息输出为csv文件，请用以下命令

```
set OutPath [输出路径]
```

##### 注意项：

* 输出文件中`\n`换行字符被替换成了`+`字符，在后续查看中可替换回`\n`字符。

* 如果命令执行后的回显信息为`[sudo] liuxinyu 的密码：`或类似信息，则标识该项命令排查信息为空