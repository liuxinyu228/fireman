package command

import (
	"bytes"
	"fireman/internal/database"
	"fireman/internal/util"
	"fmt"
	"github.com/pkg/sftp"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type SSHConfig struct {
	IP             string
	Port           int
	User           string
	PassWord       string
	PrivateKeyFile ssh.Signer //证书认证，公钥文件地址
	Kh             knownHosts
}

type SSHClient struct {
	OsRelease   string //服务器对应的发行版本
	Config      *SSHConfig
	Client      *ssh.Client //SSH连接客户端，生成新的Session
	MessagePipe *singleWriter
}

type singleWriter struct {
	b  bytes.Buffer
	mu sync.Mutex
}

func (w *singleWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

type knownHosts struct {
	HostHashMap map[string]string
}

func CreateSSHClient(r *database.Resource) (Performer, error) {
	var key ssh.Signer
	if r.KeyFile != "" {
		key, _ = Key(r.KeyFile)
	}
	Client, err := NewSSHClient(r.IP, r.Port, r.User, r.Passwd, key)
	return Client, err
}

func NewSSHClient(IP string, Port int, User string, Password string, key ssh.Signer) (Performer, error) {
	var authMethods []ssh.AuthMethod
	var sshcli SSHClient
	var pwd []byte
	if key != nil {
		authMethods = []ssh.AuthMethod{
			ssh.PublicKeys(key),
		}
	} else {
		authMethods = []ssh.AuthMethod{
			ssh.Password(Password),
		}
	}
	config := &ssh.ClientConfig{
		User:            User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", IP, Port), config)
	if err != nil {
		return nil, fmt.Errorf("初始化SSH客户端时发生错误：%s", err)
	}
	sshcli.Client = conn
	sshcli.Config = &SSHConfig{IP: IP, Port: Port, User: User, PassWord: Password}
	sshcli.MessagePipe = &singleWriter{}
	sshcli.getOS()
	if key != nil && err == nil {
		fmt.Printf("当前使用证书认证，请再输入%s用户密码用于sudo命令：", User)
		pwd, err = terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Printf("\n输入密码发送错误:%s，会影响不能使用sudo执行命令\n", err)
		}
		fmt.Println()
		sshcli.Config.PassWord = string(pwd)
	}
	return &sshcli, nil
}

func Key(keyFilePath string) (ssh.Signer, error) {
	//用于获取SSH用于认证的公钥文件
	if !util.FileExists(keyFilePath) {
		return nil, fmt.Errorf("文件不存在或文件权限不足")
	}
	key, _ := os.ReadFile(keyFilePath)

	// 解析私钥

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		if _, ok := err.(*ssh.PassphraseMissingError); ok {
			fmt.Printf("%s证书需要密码，请输入密码：", keyFilePath)
			pwd, err := terminal.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return nil, err
			}
			fmt.Println()
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, pwd)
			if err != nil {
				return nil, err
			}
			return signer, nil
		}
		return nil, fmt.Errorf("解析私钥文件时发送错误：%s", err)
	}

	return signer, nil
}

func (mc SSHClient) Actuator(command string) ([]byte, error) {
	if mc.Client == nil {
		return []byte{}, fmt.Errorf("无初始化SSH客户端，无法创建新的会话")
	}
	session, err := mc.Client.NewSession()
	if err != nil {
		return []byte{}, fmt.Errorf("%s初始化会话发生错误：%s", fmt.Sprintf("%s:%d:%s", mc.Config.IP, mc.Config.Port, mc.Config.User), err)
	}
	defer session.Close()
	session.Stdout = mc.MessagePipe
	session.Stderr = mc.MessagePipe

	err = session.Run(command)
	if err != nil {
		return []byte{}, fmt.Errorf("执行命令：%s发送错误:%s", command, err)
	}
	date := mc.MessagePipe.b.Bytes()
	mc.MessagePipe.b.Reset()
	return date, nil
}

func (mc SSHClient) Ping() bool {
	session, err := mc.Client.NewSession()
	if err != nil {
		return false
	}
	defer session.Close()

	_, err = session.CombinedOutput("echo 'Ping'")
	if err != nil {
		return false
	}

	return true
}

func (mc *SSHClient) NewSession() (*ssh.Session, error) {
	if mc.Client == nil {
		return nil, fmt.Errorf("无初始化SSH客户端，无法创建新的会话")
	}
	session, err := mc.Client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("创建新的会话时发送错误：%s", err)
	}
	return session, nil
}

func (mc *SSHClient) Close() {
	mc.Client.Close()
}

func (mc *SSHClient) GetError() []byte {
	//在执行报错时才会用
	msg := mc.MessagePipe.b.Bytes()
	mc.MessagePipe.b.Reset()
	return msg
}

func (mc *SSHClient) getOS() {
	var name []byte
	var err error
	name, err = mc.Actuator("cat /etc/os-release")
	if err != nil {
		_ = mc.GetError()
		name, err = mc.Actuator("uname -a")
		if err != nil {
			_ = mc.GetError()
			mc.OsRelease = "CENTOS"
		}
	}

	re := regexp.MustCompile(`(?i)(centos|ubuntu|debian|red hat|redhat|opensuse)`)
	match := re.FindString(string(name))
	if match == "" {
		match = "CENTOS"
	}
	mc.OsRelease = strings.ToUpper(util.Strip(match))
}

func (mc *SSHClient) OSName() string {
	return mc.OsRelease
}

func (mc *SSHClient) SudoActuator(command string) ([]byte, error) {
	//使用sudo命令尝试对命令进行执行
	var msg []byte
	tem := fmt.Sprintf("cmd=$(echo \"%s\" | sudo -S bash -c \"%s\");echo \"cmd:$cmd\"", mc.Config.PassWord, command)
	session, err := mc.Client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	//modes := ssh.TerminalModes{
	//	ssh.ECHO:          0,     // disable echoing
	//	ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
	//	ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	//}
	//
	//err = session.RequestPty("xterm", 80, 40, modes)
	//if err != nil {
	//	return []byte{}, err
	//}

	//in, err := session.StdinPipe()
	//if err != nil {
	//	return nil, err
	//}

	session.Stderr = mc.MessagePipe
	session.Stdout = mc.MessagePipe
	//err = session.Start(tem)
	//if err != nil {
	//	return []byte{}, err
	//}
	////in.Write([]byte(mc.Config.PassWord + "\n"))
	//err = session.Wait()
	//if err != nil {
	//	return []byte{}, err
	//}
	//separator := []byte("cmd:")
	//l := bytes.Split(mc.MessagePipe.b.Bytes(), separator)
	//mc.MessagePipe.b.Reset()
	//if len(l) >= 2 {
	//	msg = l[0]
	//	if util.Strip(string(l[1])) != "" {
	//		msg = l[1]
	//	}
	//}
	err = session.Run(tem)
	if err != nil {
		return []byte{}, fmt.Errorf("执行命令：%s发送错误:%s", command, err)
	}
	date := mc.MessagePipe.b.Bytes()
	mc.MessagePipe.b.Reset()
	separator := []byte("cmd:")
	l := bytes.Split(date, separator)
	if len(l) >= 2 {
		msg = l[0]
		if util.Strip(string(l[1])) != "" {
			msg = l[1]
		}
	}
	return msg, nil
}

func (mc *SSHClient) UploadFile(localPath, remotePath string) (int64, error) {
	barErrorChan := make(chan struct{})
	sftpClient, err := sftp.NewClient(mc.Client)
	if err != nil {
		return 0, err
	}
	defer sftpClient.Close()

	srcFile, err := os.Open(localPath)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()

	rinfo, err := sftpClient.Stat(remotePath)
	if err == nil {
		if rinfo.IsDir() {
			remotePath = path.Join(remotePath, filepath.Base(srcFile.Name()))
			_, err = sftpClient.Stat(remotePath)
		}
		if err == nil {
			var Y string
			fmt.Printf("远程文件%s已经存在是否覆盖(Y/N): ", remotePath)
			fmt.Scanln(&Y)
			if strings.ToUpper(Y) != "Y" {
				return 0, os.ErrExist
			}
		}
	}

	dstFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()

	// 获取本地文件大小
	fileInfo, err := srcFile.Stat()
	if err != nil {
		return 0, err
	}

	// 创建进度条
	bar := progressbar.DefaultBytesSilent(
		fileInfo.Size(),
		"upload:")
	var wg sync.WaitGroup
	wg.Add(1)
	go func(br chan struct{}) {
	outfor:
		for {
			select {
			case <-br:
				break outfor
			default:
				msg := bar.String()
				if msg != "" {
					fmt.Printf("\r%s\r", strings.TrimSpace(msg))
				}
				if bar.IsFinished() {
					msg = bar.String()
					if msg != "" {
						fmt.Printf("\r%s\r", strings.TrimSpace(msg))
					}
					break outfor
				}
			}
		}
		fmt.Println()
		wg.Done()
	}(barErrorChan)
	// 从本地文件复制到远程文件，同时更新进度条
	i, err := dstFile.ReadFrom(io.TeeReader(srcFile, bar))
	if err != nil {
		barErrorChan <- struct{}{}
		return 0, err
	}
	wg.Wait()
	return i, nil
}

func (mc *SSHClient) DownloadFile(remotePath string, localPath string) (int64, error) {
	var sum int64
	barErrorChan := make(chan struct{})
	sftpClient, err := sftp.NewClient(mc.Client)
	if err != nil {
		return 0, err
	}
	defer sftpClient.Close()

	srcFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()

	rinfo, err := os.Stat(localPath)
	if err == nil {
		if rinfo.IsDir() {
			localPath = path.Join(localPath, filepath.Base(srcFile.Name()))
			_, err = os.Stat(localPath)
		}
		if err == nil {
			var Y string
			fmt.Printf("本地文件%s已经存在是否覆盖(Y/N): ", localPath)
			fmt.Scanln(&Y)
			if strings.ToUpper(Y) != "Y" {
				return 0, os.ErrExist
			}
		}
	}

	dstFile, err := os.Create(localPath)
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()

	// 获取远程文件大小
	fileInfo, err := srcFile.Stat()
	if err != nil {
		return 0, err
	}

	// 创建进度条
	bar := progressbar.DefaultBytesSilent(
		fileInfo.Size(),
		"download:")

	var wg sync.WaitGroup
	wg.Add(1)
	go func(br chan struct{}) {
	outfor:
		for {
			//fmt.Println(1)
			select {
			case <-br:
				break outfor
			default:
				msg := bar.String()
				if msg != "" {
					fmt.Printf("\r%s\r", strings.TrimSpace(msg))
				}
				if bar.IsFinished() {
					msg = bar.String()
					if msg != "" {
						fmt.Printf("\r%s\r", strings.TrimSpace(msg))
					}
					break outfor
				}
			}

		}
		fmt.Println()
		wg.Done()
	}(barErrorChan)

	// 从远程文件复制到本地文件，同时更新进度条
	//i, err := io.Copy(io.MultiWriter(dstFile, bar), srcFile)
	//if err != nil {
	//	return 0, err
	//}
	buf := make([]byte, 1024000)
	for {
		n, err := srcFile.Read(buf)
		dstFile.Write(buf[:n])
		bar.Write(buf[:n])
		sum += int64(n)
		if err != nil {
			if err == io.EOF {
				break
			}
			barErrorChan <- struct{}{}
			return sum, err
		}
	}
	wg.Wait()
	return sum, nil
}
