package controller

import (
	"errors"
	"riff/logic"
	"riff/models"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// VoteHandler 为帖子投票或取消投票。
// @Summary 帖子投票
// @Tags 投票
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body models.ParamVote true "direction 只能为 -1、0、1"
// @Success 200 {object} ResponseData
// @Router /vote [post]
func VoteHandler(c *gin.Context) {
	//参数校验
	p := new(models.ParamVote)
	if err := c.ShouldBindJSON(p); err != nil {
		zap.L().Error("ShouldBindJSON failed", zap.Error(err))
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			ResponseError(c, CodeInvalidParam)
			return
		}

		errData := removeTopStruct(errs.Translate(trans))
		ResponseErrorWithMsg(c, CodeInvalidParam, errData)
		return
	}

	//获取当前请求的用户的id
	userID, err := getCurrentUser(c)
	if err != nil {
		ResponseError(c, CodeNeedLogin)
		return
	}
	//具体的投票业务逻辑
	if err := logic.VoteForPost(c.Request.Context(), userID, p); err != nil {
		zap.L().Error("logic.VoteForPost() failed", zap.Error(err))
		switch {
		case errors.Is(err, logic.ErrPostNotExist):
			ResponseError(c, CodePostNotExist)
		case errors.Is(err, logic.ErrVoteTimeExpire):
			ResponseError(c, CodeVoteTimeExpire)
		case errors.Is(err, logic.ErrInvalidVote):
			ResponseError(c, CodeInvalidParam)
		default:
			ResponseError(c, CodeServerBusy)
		}
		return
	}

	ResponseSuccess(c, nil)
}
