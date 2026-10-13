package mysql

import (
	"errors"
	"riff/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func GetCommunityList() (communityList []*models.Community, err error) {

	err = db.
		Select("community_id", "community_name", "introduction").
		Find(&communityList).Error

	if err != nil {
		zap.L().Error("query community failed", zap.Error(err))
		return
	}

	return
}

func GetCommunityDetailByID(id int64) (community *models.Community, err error) {

	err = db.
		Select("community_id", "community_name", "introduction", "create_time").
		Where("community_id = ?", id).
		First(&community).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrorCommunityNotExist
	}
	if err != nil {
		return nil, err
	}

	return
}
