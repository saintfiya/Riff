package logic

import (
	"context"
	"errors"
	"sort"
	"riff/dao/mysql"
	"riff/dao/redis"
	"riff/models"
	"riff/pkg/snowflake"
)

func CreatePost(ctx context.Context, p *models.Post) error {
	//生成PostID
	p.PostID = snowflake.GenID()

	//保存到数据库
	err := mysql.CreatePost(p)
	if err != nil {
		return err
	}

	if err = redis.CreatePost(ctx, p.PostID); err != nil {
		// MySQL 已成功、Redis 失败时进行补偿，避免留下无法投票的帖子。
		if rollbackErr := mysql.DeletePost(p.PostID); rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}

	return nil
}

func PostList(page int64, size int64) (posts []*models.Post, err error) {

	posts, err = mysql.PostList(page, size)
	if err != nil {
		return nil, err
	}
	return
}

// GetPostList2 返回按 Redis 排行、并由 MySQL 补齐作者与社区信息的帖子列表。
func GetPostList2(ctx context.Context, params *models.ParamPostList) ([]*models.ApiPostDetail, error) {
	var (
		ids []int64
		err error
	)
	if params.CommunityID == 0 {
		ids, err = redis.GetPostIDsInOrder(ctx, params)
	} else {
		candidateIDs, queryErr := mysql.GetPostIDsByCommunity(params.CommunityID)
		if queryErr != nil {
			return nil, queryErr
		}
		scores, scoreErr := redis.GetPostOrderScores(ctx, candidateIDs, params.Order)
		if scoreErr != nil {
			return nil, scoreErr
		}
		ids = rankAndPagePostIDs(candidateIDs, scores, params.Page, params.Size)
	}
	if err != nil {
		return nil, err
	}

	details, err := mysql.GetPostListByIDs(ids)
	if err != nil {
		return nil, err
	}
	detailIDs := make([]int64, 0, len(details))
	for _, detail := range details {
		detailIDs = append(detailIDs, detail.PostID)
	}
	votes, err := redis.GetPostVoteData(ctx, detailIDs)
	if err != nil {
		return nil, err
	}
	for _, detail := range details {
		detail.VoteNum = votes[detail.PostID]
	}
	return details, nil
}

func rankAndPagePostIDs(ids []int64, scores map[int64]float64, page, size int64) []int64 {
	ranked := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := scores[id]; ok {
			ranked = append(ranked, id)
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		if scores[ranked[i]] == scores[ranked[j]] {
			return ranked[i] > ranked[j]
		}
		return scores[ranked[i]] > scores[ranked[j]]
	})

	start := (page - 1) * size
	if start >= int64(len(ranked)) {
		return []int64{}
	}
	end := start + size
	if end > int64(len(ranked)) {
		end = int64(len(ranked))
	}
	return ranked[start:end]
}

func GetPostByID(pid int64) (post *models.Post, err error) {

	post, err = mysql.GetPostByID(pid)
	if err != nil {
		if errors.Is(err, mysql.ErrorPostNotExist) {
			return nil, ErrPostNotExist
		}
		return nil, err
	}
	return
}
