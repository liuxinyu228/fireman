/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fireman/internal/database"
	"fireman/internal/util"
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"log"
	"os"
	"strconv"
	"strings"
)

var SourceFile string

var qvIP string
var qvUser string
var queryALL bool

var upUser string
var upPwd string
var upkeyFile string

var delID int

// resCmd represents the res command
var resCmd = &cobra.Command{
	Use:   "res",
	Short: "资源信息处理",
	Long:  `add用于添加资源,query用于查询资源信息`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

// 资源添加模块
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "添加资源信息",
	Long:  "添加资源IP、Port、User、Passwd等信息,其中User、Passwd加密存储",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := database.NewDB()
		if err != nil {
			util.PrintErr(err)
		}
		if SourceFile != "" {
			addResourceForFile(db, SourceFile)
		} else {
			addResourceShell(db)
		}
	},
}

// 资源查询模块
var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "用于查询展示资源信息",
	Run: func(cmd *cobra.Command, args []string) {
		var result []database.Resource
		db, err := database.NewDB()
		if err != nil {
			util.PrintErr(err)
		}
		if (qvIP == "" && qvUser == "") && !queryALL {
			util.PrintErr(fmt.Errorf("请输入查询条件，可多个条件进行查询"))
		}
		if queryALL {
			db.Find(&result)
		} else {
			result = queryResource(db, qvIP)
		}

		for _, r := range result {
			if qvUser != "" {
				if r.User != qvUser {
					continue
				}
			}
			fmt.Printf("ID:%d\t%s\n", r.ID, r.String())
		}
	},
}

// 资源修改模块
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "用于更新资源信息",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := database.NewDB()
		if err != nil {
			util.PrintErr(err)
		}
		if updateResource(db, qvIP, upUser, upPwd, upkeyFile) {
			fmt.Printf("资源%s修改成功", qvIP)
		} else {
			fmt.Printf("资源%s不存在或者已经存在", qvIP)
		}
	},
}

var delCmd = &cobra.Command{
	Use:   "del",
	Short: "用于删除资源信息",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := database.NewDB()
		if err != nil {
			util.PrintErr(err)
		}
		result := db.Delete(&database.Resource{ID: delID})
		if result.RowsAffected > 0 {
			fmt.Println("删除成功！")
		} else {
			fmt.Println("删除失败！")
		}

	},
}

func init() {
	updateCmd.Flags().StringVarP(&upUser, "user", "u", "", "查询的user值")
	updateCmd.Flags().StringVarP(&upPwd, "passwd", "w", "", "修改的passwd值，置空密码 -w \" \"")
	updateCmd.Flags().StringVarP(&upkeyFile, "keyfile", "k", "", "修改的keyfile值")

	queryCmd.Flags().StringVarP(&qvUser, "qvUser", "u", "", "查询的User值,只能和qvID和qvIP配合使用")
	queryCmd.Flags().BoolVarP(&queryALL, "all", "a", false, "查询所有数据")

	addCmd.Flags().StringVarP(&SourceFile, "file", "f", "", "批量插入资源信息的模板信息")

	delCmd.Flags().IntVarP(&delID, "id", "d", -1, "删除指定ID的资源")

	rootCmd.PersistentFlags().StringVarP(&qvIP, "ip", "i", "", "查询的IP值")
	resCmd.AddCommand(addCmd)
	resCmd.AddCommand(queryCmd)
	resCmd.AddCommand(updateCmd)
	resCmd.AddCommand(delCmd)
	rootCmd.AddCommand(resCmd)
}

func addResourceShell(db *gorm.DB) {
	fmt.Println("Adding a new resource...")

	// 创建一个 bufio.Scanner 以从标准输入中获取用户输入
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("请输入资源名称: ")
	scanner.Scan()
	name := util.Strip(scanner.Text())

	fmt.Print("请输入IP地址: ")
	scanner.Scan()
	ip := util.Strip(scanner.Text())

	fmt.Print("请输入端口: ")
	scanner.Scan()
	port := util.Strip(scanner.Text())
	uport, err := strconv.Atoi(port)
	if err != nil {
		util.PrintErr(fmt.Errorf("请输入正确的端口数值！"))
	}

	fmt.Print("请输入连接协议: ")
	scanner.Scan()
	protocol := strings.ToUpper(util.Strip(scanner.Text()))

	fmt.Print("请输入用户: ")
	scanner.Scan()
	user := util.Strip(scanner.Text())

	fmt.Print("请输入密码: ")
	scanner.Scan()
	passwd := util.Strip(scanner.Text())

	fmt.Print("请输入SSH公钥文件路径，如果不使用证书认证请按回车: ")
	scanner.Scan()
	filePath := util.Strip(scanner.Text())

	fmt.Println("添加资源信息如下：")
	fmt.Printf("资源名称：\t%s\n资源IP：\t%s\n资源端口：\t%s\n资源协议：\t%s\n登陆用户：\t%s\n登陆密码：\t%s\n登陆证书：\t%s\n", name, ip, port, protocol, user, passwd, filePath)
	fmt.Print("确定插入资源(Y/N)：")
	scanner.Scan()
	o := util.Strip(scanner.Text())
	if strings.ToUpper(o) == "Y" {
		err = database.CreateResource(db, name, ip, uport, protocol, user, passwd, filePath)
		if err == nil {
			fmt.Printf("插入资源{%s}信息成功", ip)
		} else {
			util.PrintErr(err)
		}
	} else {
		fmt.Println("用户取消操作！")
	}
}

func addResourceForFile(db *gorm.DB, sourceFile string) error {
	if !util.FileExists(sourceFile) {
		return os.ErrNotExist
	}
	// 打开配置文件
	configFile, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer configFile.Close()

	var resList map[string]database.Resource
	decoder := yaml.NewDecoder(configFile)
	if err := decoder.Decode(&resList); err != nil {
		return err
	}

	for name, res := range resList {
		res.Name = name
		res.Protocol = strings.ToUpper(res.Protocol)
		if err := res.CheckValue(); err != nil {
			log.Println(err)
			continue
		}
		err := database.CreateResource(db, res.Name, res.IP, res.Port, res.Protocol, res.User, res.Passwd, res.KeyFile)
		if err != nil {
			if err == os.ErrExist {
				log.Printf("IP:%s,Port:%d,User:%s信息已经在数据库中...", res.IP, res.Port, res.User)
				continue
			}
			return err
		}
		log.Println("资源插入成功：", res.Name)
	}

	return nil
}

func queryResource(db *gorm.DB, ip string) []database.Resource {
	var result []database.Resource
	db.Where(&database.Resource{IP: ip}).Find(&result)
	return result
}

func updateResource(db *gorm.DB, ip string, user string, passwd string, pkf string) bool {
	var Plist []database.Resource
	passwd = strings.ReplaceAll(passwd, "\"", "")
	pkf = strings.ReplaceAll(pkf, "\"", "")
	qr := database.Resource{IP: ip, User: user, Passwd: passwd}
	db.Model(&qr).Where("ip = ?", ip).Find(&Plist)
	ID := database.GetResourceID(Plist, qr)
	unpwd, _ := util.GetEncryptData(passwd)
	result := db.Model(&qr).Where("id = ?", ID).Updates(&database.Resource{Passwd: unpwd, KeyFile: pkf})
	if result.RowsAffected > 0 {
		return true
	}
	return false
}
