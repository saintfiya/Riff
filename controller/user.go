package controller

import (
	"errors"
	"net/http"
	"riff/logic"
	"riff/models"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// SignUpHandler 用户注册。
// @Summary 用户注册
// @Tags 用户
// @Accept json
// @Produce json
// @Param body body models.ParamSignUp true "注册信息"
// @Success 200 {object} ResponseData
// @Failure 400 {object} object
// @Router /signup [post]
func SignUpHandler(ctx *gin.Context) {
	//1.获取参数和参数校验
	p := new(models.ParamSignUp)
	if err := ctx.ShouldBindJSON(p); err != nil {
		zap.L().Error("ctx.ShouldBindJSON failed", zap.Error(err))
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"err": err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{
			"err": errs.Translate(trans),
		})

		return
	}
	//2.业务处理
	if err := logic.SignUp(p); err != nil {
		if errors.Is(err, logic.ErrUserExist) {
			ResponseError(ctx, CodeUserExist)
			return
		}
		ResponseError(ctx, CodeServerBusy)
		return

	}

	//3.返回响应
	ResponseSuccess(ctx, nil)
}

// LoginHandler 用户登录并返回 JWT。
// @Summary 用户登录
// @Tags 用户
// @Accept json
// @Produce json
// @Param body body models.ParamLogin true "登录信息"
// @Success 200 {object} ResponseData
// @Failure 400 {object} object
// @Router /login [post]
func LoginHandler(ctx *gin.Context) {
	//获取参数和参数校验
	p := new(models.ParamLogin)
	if err := ctx.ShouldBindJSON(p); err != nil {
		zap.L().Error("ctx.ShouldBindJSON failed", zap.Error(err))
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"err": err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{
			"err": errs.Translate(trans),
		})

		return
	}
	//业务处理
	user, err := logic.Login(p)
	if err != nil {
		switch {
		case errors.Is(err, logic.ErrUserNotExist):
			ResponseError(ctx, CodeUserNotExist)
		case errors.Is(err, logic.ErrInvalidPassword):
			ResponseError(ctx, CodeInvalidPassword)
		default:
			zap.L().Error("logic.Login failed", zap.Error(err))
			ResponseError(ctx, CodeServerBusy)
		}
		return

	}
	//返回响应
	ResponseSuccess(ctx, user.Token)
}
