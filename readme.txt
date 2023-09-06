批量应急响应工具fireman
1.通过fireman res模块添加资源信息，资源信息模板为Source.yaml,会初始化db文件到./static/data.db中
2.通过fireman res update更新资源信息时，如果要将密码或者证书置为空，使用 -w " "或者-k " "
3.数据库中的user和passwd字段为加密字段
4.通过fireman run模块进行运行命令,所有命令均有提示信息。checkALL为一键排查所有步骤的命令
5.暂时只支持linux系统, 设置服务器通过证书登陆教程：https://www.runoob.com/w3cnote/set-ssh-login-key.html