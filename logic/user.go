package logic

import (
	"riff/dao/mysql"
	"riff/models"
	"riff/pkg/jwt"
	"riff/pkg/snowflake"
)

var (
	ErrUserExist       = mysql.ErrorUserExist
	ErrUserNotExist    = mysql.ErrorUserNotExist
	ErrInvalidPassword = mysql.ErrorInvalidPassword
)

func SignUp(p *models.ParamSignUp) (err error) {
	//生成UID
	userID := snowflake.GenID()

	//构造一个用户实例
	user := models.User{
		UserID:   userID,
		Username: p.Username,
		Password: p.Password,
	}
	//密码加密

	if err := mysql.InsertUser(&user); err != nil {
		return err
	}

	return nil
}

func Login(p *models.ParamLogin) (user *models.User, err error) {
	user = &models.User{
		Username: p.Username,
		Password: p.Password,
	}
	//查询数据库
	if err := mysql.Login(user); err != nil {
		return nil, err
	}

	//生成jwt
	token, err := jwt.GenToken(user.UserID, user.Username)
	if err != nil {
		return nil, err
	}
	user.Token = token
	return
}
