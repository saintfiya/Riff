package controller

import (
	"errors"
	"strconv"
	"riff/logic"
	"riff/models"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// CreatePost 发布一个帖子。
// @Summary 发布帖子
// @Tags 帖子
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body models.Post true "帖子"
// @Success 200 {object} ResponseData
// @Router /post [post]
func CreatePost(ctx *gin.Context) {
	//获取参数和参数校验
	p := new(models.Post)
	if err := ctx.ShouldBindJSON(p); err != nil {
		zap.L().Error("ctx.ShouldBindJSON failed", zap.Error(err))
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			ResponseError(ctx, CodeInvalidParam)
			return
		}
		ResponseErrorWithMsg(ctx, CodeInvalidParam, removeTopStruct(errs.Translate(trans)))
		return
	}
	//从c取到当前发请求的用户的ID
	userID, err := getCurrentUser(ctx)
	if err != nil {
		zap.L().Error("getCurrentUser failed", zap.Error(err))
		ResponseError(ctx, CodeNeedLogin)
		return
	}

	p.AuthorID = userID
	//业务处理
	if err := logic.CreatePost(ctx.Request.Context(), p); err != nil {
		zap.L().Error("logic.CreatePost failed", zap.Error(err))
		ResponseError(ctx, CodeServerBusy)
		return
	}

	//返回响应
	ResponseSuccess(ctx, nil)
}

// PostList 是旧版的纯 MySQL 列表实现。
// @Summary 帖子分页查询接口
// @Description 按发布时间分页查询帖子列表
// @Tags 帖子
// @Accept application/json
// @Produce application/json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Success 200 {object} _ResponsePostList
// @Router /posts [get]
func PostList(ctx *gin.Context) {
	//获取分页参数
	page, size := getPageInfo(ctx)
	posts, err := logic.PostList(page, size)
	if err != nil {
		zap.L().Error("logic.PostList failed", zap.Error(err))
		ResponseError(ctx, CodeServerBusy)
		return
	}

	//返回响应
	ResponseSuccess(ctx, posts)
}

// GetPostByID 查询某个帖子。
// @Summary 查询帖子详情
// @Tags 帖子
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "帖子 ID"
// @Success 200 {object} ResponseData
// @Router /post/{id} [get]
func GetPostByID(ctx *gin.Context) {
	//从路径参数中获取PostID
	pidStr := ctx.Param("id")

	pid, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil {
		zap.L().Error("strconv.ParseInt failed", zap.Error(err))
		ResponseError(ctx, CodeInvalidParam)
		return
	}

	//根据id取出数据
	post, err := logic.GetPostByID(pid)
	if err != nil {
		zap.L().Error("logic.GetPostByID failed", zap.Error(err))
		if errors.Is(err, logic.ErrPostNotExist) {
			ResponseError(ctx, CodePostNotExist)
		} else {
			ResponseError(ctx, CodeServerBusy)
		}
		return
	}

	//返回响应
	ResponseSuccess(ctx, post)
}

// GetPostListHandler2 返回带作者、社区和净投票数的帖子列表。
// @Summary 升级版帖子列表接口
// @Description 可分页并按发布时间或热度排序，也可按社区过滤
// @Tags 帖子
// @Accept application/json
// @Produce application/json
// @Param object query models.ParamPostList false "查询参数"
// @Success 200 {object} _ResponsePostDetailList
// @Router /posts2 [get]
func GetPostListHandler2(ctx *gin.Context) {
	params := &models.ParamPostList{
		Page:  1,
		Size:  10,
		Order: models.OrderTime,
	}
	if err := ctx.ShouldBindQuery(params); err != nil {
		zap.L().Error("ctx.ShouldBindQuery failed", zap.Error(err))
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			ResponseError(ctx, CodeInvalidParam)
			return
		}
		ResponseErrorWithMsg(ctx, CodeInvalidParam, removeTopStruct(errs.Translate(trans)))
		return
	}

	data, err := logic.GetPostList2(ctx.Request.Context(), params)
	if err != nil {
		zap.L().Error("logic.GetPostList2 failed", zap.Error(err))
		ResponseError(ctx, CodeServerBusy)
		return
	}
	ResponseSuccess(ctx, data)
}
