package logic

import (
	"errors"
	"riff/dao/mysql"
	"riff/models"
)

func GetCommunityList() ([]*models.Community, error) {
	return mysql.GetCommunityList()
}

func GetCommunityDetailByID(id int64) (*models.Community, error) {
	community, err := mysql.GetCommunityDetailByID(id)
	if errors.Is(err, mysql.ErrorCommunityNotExist) {
		return nil, ErrCommunityNotExist
	}
	return community, err
}
