package database

import (
	"fireman/internal/config"
	"fireman/internal/util"
	"fmt"
	"gorm.io/gorm"
	"os"
	"time"
)

type Resource struct {
	ID        int    `gorm:"primaryKey"`
	Name      string `yaml:"name"`
	IP        string `gorm:"type:varchar(20)";yaml:"ip"`
	Port      int    `yaml:"port"`
	Protocol  string `yaml:"protocol"`
	User      string `gorm:"type:text";yaml:"user"`
	Passwd    string `gorm:"type:text";yaml:"passwd"`
	KeyFile   string `gorm:"type:text";yaml:"keyfile"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (r *Resource) String() string {
	return fmt.Sprintf("Name:%s\tIP:%s\tPort:%d\tProtocol:%s\tUser:%s\tPasswd:%s\tKeyFile:%s\t\n", r.Name, r.IP, r.Port, r.Protocol, r.User, r.Passwd, r.KeyFile)
}

func (r *Resource) CheckValue() error {
	if !util.IsValidIP(r.IP) {
		return fmt.Errorf("资源%s,ip值填写错误", r.Name)
	}
	if r.Protocol != "SSH" && r.Protocol != "WINRM" {
		return fmt.Errorf("资源%s,protocol值填写错误，只支持SSH/WINRM", r.Name)
	}
	if r.Port > 65535 || r.Port < 0 {
		return fmt.Errorf("资源%s,port值填写错误，端口值为0-65535", r.Name)
	}
	if !util.FileExists(r.KeyFile) && r.KeyFile != "" {
		return fmt.Errorf("资源%s,keyfile值填写错误，文件{%s}不存在或者无权限", r.Name, r.KeyFile)
	}
	return nil
}

func (r *Resource) BeforeCreate(tx *gorm.DB) (err error) {
	var count int64
	var Plist []*Resource

	// 创建 Resource 对象作为查询条件
	queryResource := Resource{
		IP:   r.IP,
		Port: r.Port,
	}
	result := tx.Model(&Resource{}).Where(&queryResource).Find(&Plist)
	if result.Error != nil {
		fmt.Printf("获取当前数据{%s}总数报错：%s\n", r.IP, result.Error)
		return result.Error
	}
	if len(Plist) >= 1 {
		//ip.port.user.passwd四个条件只能同时存在一条
		if IsrepeatResource(r, Plist) {
			return os.ErrExist
		}

	}

	tx.Model(&Resource{}).Count(&count)
	r.ID = int(count + 1)
	enUser, err := util.Encrypt(config.SecretKey, []byte(r.User))
	if err != nil {
		fmt.Printf("对数据{%s}进行加密时发送错误: %s", r.User, err)
		return err
	}
	enPwd, err := util.Encrypt(config.SecretKey, []byte(r.Passwd))
	if err != nil {
		fmt.Printf("对数据{%s}进行加密时发送错误: %s", r.Passwd, err)
		return err
	}
	r.User = util.ByteToBase64(enUser)
	r.Passwd = util.ByteToBase64(enPwd)

	return nil
}

func (r *Resource) AfterFind(tx *gorm.DB) (err error) {
	//查询时进行解密操作
	user, _ := util.Decrypt(config.SecretKey, util.Base64DecodeToByte(r.User))
	pwd, _ := util.Decrypt(config.SecretKey, util.Base64DecodeToByte(r.Passwd))
	r.User = string(user)
	r.Passwd = string(pwd)
	return nil
}

func IsrepeatResource(NR *Resource, Plist []*Resource) bool {
	// 判断是否有重复的资源信息，user:passwd或者user:pkeyFile只能唯一

	for _, r := range Plist {
		if (NR.User == r.User && NR.Passwd == r.Passwd) || (NR.User == r.User && NR.KeyFile == r.KeyFile) {
			return true
		}
	}

	return false
}
