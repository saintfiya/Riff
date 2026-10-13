package mysql

import (
	"errors"
	"riff/models"

	"gorm.io/gorm"
)

func CreatePost(p *models.Post) error {
	err := db.
		Create(p).
		Error

	if err != nil {

		return err
	}
	return nil
}

// DeletePost 删除帖子
func DeletePost(postID int64) error {
	return db.Where("post_id = ?", postID).Delete(&models.Post{}).Error
}

// GetPostList 查询帖子列表
func PostList(page, size int64) (posts []*models.Post, err error) {
	err = db.
		Select("post_id, title, content, author_id, community_id, create_time").
		Order("create_time DESC").
		Offset(int((page - 1) * size)).
		Limit(int(size)).
		Find(&posts).Error

	if err != nil {
		return nil, err
	}
	return
}

// GetPostIDsByCommunity 返回指定社区中所有可见帖子的 ID。
func GetPostIDsByCommunity(communityID int64) ([]int64, error) {
	var ids []int64
	err := db.Model(&models.Post{}).
		Where("community_id = ? AND status = ?", communityID, 1).
		Pluck("post_id", &ids).Error
	return ids, err
}

// GetPostListByIDs 批量加载帖子及关联的作者、社区，并保持 ids 的顺序。
func GetPostListByIDs(ids []int64) ([]*models.ApiPostDetail, error) {
	if len(ids) == 0 {
		return []*models.ApiPostDetail{}, nil
	}

	var posts []*models.Post
	if err := db.Where("post_id IN ? AND status = ?", ids, 1).Find(&posts).Error; err != nil {
		return nil, err
	}
	if len(posts) == 0 {
		return []*models.ApiPostDetail{}, nil
	}

	authorIDs := make([]int64, 0, len(posts))
	communityIDs := make([]int64, 0, len(posts))
	postsByID := make(map[int64]*models.Post, len(posts))
	for _, post := range posts {
		postsByID[post.PostID] = post
		authorIDs = append(authorIDs, post.AuthorID)
		communityIDs = append(communityIDs, post.CommunityID)
	}

	var users []*models.User
	if err := db.Select("user_id", "username").Where("user_id IN ?", authorIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	usersByID := make(map[int64]string, len(users))
	for _, user := range users {
		usersByID[user.UserID] = user.Username
	}

	var communities []*models.Community
	if err := db.Where("community_id IN ?", communityIDs).Find(&communities).Error; err != nil {
		return nil, err
	}
	communitiesByID := make(map[int64]*models.Community, len(communities))
	for _, community := range communities {
		communitiesByID[int64(community.CommunityID)] = community
	}

	details := make([]*models.ApiPostDetail, 0, len(posts))
	for _, id := range ids {
		post, ok := postsByID[id]
		if !ok {
			continue
		}
		details = append(details, &models.ApiPostDetail{
			AuthorName: usersByID[post.AuthorID],
			Post:       post,
			Community:  communitiesByID[post.CommunityID],
		})
	}
	return details, nil
}

func GetPostByID(pid int64) (post *models.Post, err error) {
	err = db.
		Where("post_id = ?", pid).
		First(&post).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrorPostNotExist
	}
	if err != nil {
		return nil, err
	}
	return
}
