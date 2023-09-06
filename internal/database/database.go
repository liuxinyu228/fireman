package database

import (
	"fireman/internal/config"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

func NewDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(config.DB), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("初始化数据库发生错误：%s", err)
	}
	//自动迁移对象
	db.AutoMigrate(&Resource{})
	return db, nil
}

// CreateResource 创建资源
func CreateResource(db *gorm.DB, name string, ip string, port int, protocol string, user string, passwd string, privateKeyFile string) error {
	resource := Resource{
		Name:      name,
		IP:        ip,
		Port:      port,
		Protocol:  protocol,
		User:      user,
		Passwd:    passwd,
		KeyFile:   privateKeyFile,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := resource.CheckValue(); err != nil {
		return err
	}
	return db.Create(&resource).Error
}

// GetResourceByID 查询资源
func GetResourceByID(db *gorm.DB, id int) (*Resource, error) {
	var resource Resource
	err := db.First(&resource, id).Error
	return &resource, err
}

// GetResourceByIP 根据IP查询资源
func GetResourceByIP(db *gorm.DB, ip string) ([]*Resource, error) {
	Plist := make([]*Resource, 1)
	err := db.Where("ip = ?", ip).Find(&Plist).Error
	return Plist, err
}

func GetResourceAll(db *gorm.DB) ([]*Resource, error) {
	Plist := make([]*Resource, 1)
	err := db.Find(&Plist).Error
	return Plist, err
}

// UpdateResource 更新资源
func UpdateResource(db *gorm.DB, id int, ip string, port int, protocol string, user string, passwd string, privateKeyFile string) error {
	var resource Resource
	err := db.First(&resource, id).Error
	if err != nil {
		return err
	}

	resource.IP = ip
	resource.Port = port
	resource.Protocol = protocol
	resource.User = user
	resource.Passwd = passwd
	resource.KeyFile = privateKeyFile
	resource.UpdatedAt = time.Now()

	return db.Save(&resource).Error
}

// DeleteResource 删除资源
func DeleteResource(db *gorm.DB, id int) error {
	return db.Delete(&Resource{}, id).Error
}

func GetResourceID(Plist []Resource, qr Resource) int {
	for _, p := range Plist {
		if p.IP == qr.IP && p.User == qr.User {
			if p.Passwd == qr.Passwd {
				return -1
			}
			return p.ID
		}
	}
	return -1
}
