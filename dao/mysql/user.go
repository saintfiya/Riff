package mysql

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"riff/models"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// sercret 保留用于验证旧版本密码；新密码统一使用 bcrypt。
const sercret = "saintfiya"

func InsertUser(user *models.User) error {
	password, err := hashPassword(user.Password)
	if err != nil {
		return err
	}
	user.Password = password

	if err := db.Create(user).Error; err != nil {
		var mysqlErr *mysqlDriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return ErrorUserExist
		}
		return err
	}
	return nil
}

// legacyPasswordHash 仅用于兼容升级旧数据。
func legacyPasswordHash(password string) string {
	h := md5.New()
	h.Write([]byte(sercret))
	return hex.EncodeToString(h.Sum([]byte(password)))
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func verifyPassword(storedPassword, password string) (valid, needsUpgrade bool) {
	if bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password)) == nil {
		return true, false
	}
	if bytes.Equal([]byte(legacyPasswordHash(password)), []byte(storedPassword)) {
		return true, true
	}
	return false, false
}

func Login(user *models.User) error {
	plainPassword := user.Password

	err := db.Table("users").Where("username = ?", user.Username).First(user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrorUserNotExist
	}
	if err != nil {
		return err
	}

	valid, needsUpgrade := verifyPassword(user.Password, plainPassword)
	if !valid {
		return ErrorInvalidPassword
	}
	if !needsUpgrade {
		return nil
	}

	password, err := hashPassword(plainPassword)
	if err != nil {
		return err
	}
	return db.Model(&models.User{}).
		Where("user_id = ?", user.UserID).
		Update("password", password).Error
}
