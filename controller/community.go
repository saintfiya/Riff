package controller

import (
	"errors"
	"strconv"
	"riff/logic"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CommunityList 查询社区列表。
// @Summary 查询社区列表
// @Tags 社区
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} ResponseData
// @Router /community [get]
func CommunityList(ctx *gin.Context) {
	//将查询到的所有社区(community_id,community_name)以列表的形式返回
	data, err := logic.GetCommunityList()
	if err != nil {
		zap.L().Error("logic.GetCommunityList", zap.Error(err))
		ResponseError(ctx, CodeServerBusy)
		return
	}
	//返回响应
	ResponseSuccess(ctx, data)
}

// CommunityDetail 查询某个社区详情。
// @Summary 查询社区详情
// @Tags 社区
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "社区 ID"
// @Success 200 {object} ResponseData
// @Router /community/{id} [get]
func CommunityDetail(ctx *gin.Context) {
	//从路径参数中获取社区id
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		zap.L().Error("strconv.ParseInt() failed", zap.Error(err))
		ResponseError(ctx, CodeInvalidParam)
		return
	}

	//将查询到的社区以列表的形式返回
	data, err := logic.GetCommunityDetailByID(id)
	if err != nil {
		zap.L().Error("logic.GetCommunityList", zap.Error(err))
		if errors.Is(err, logic.ErrCommunityNotExist) {
			ResponseError(ctx, CodeCommunityNotExist)
		} else {
			ResponseError(ctx, CodeServerBusy)
		}
		return
	}

	//返回响应
	ResponseSuccess(ctx, data)
}
