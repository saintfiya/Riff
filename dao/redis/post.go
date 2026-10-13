package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"riff/models"

	redisClient "github.com/redis/go-redis/v9"
)

// GetPostIDsInOrder 从 Redis 排行榜中按页返回帖子 ID。
func GetPostIDsInOrder(ctx context.Context, params *models.ParamPostList) ([]int64, error) {
	key := KeyPostTimeZset
	if params.Order == models.OrderScore {
		key = KeyPostScoreZset
	}

	start := (params.Page - 1) * params.Size
	end := start + params.Size - 1
	members, err := client.ZRevRange(ctx, getRedisKey(key), start, end).Result()
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(members))
	for _, member := range members {
		id, err := strconv.ParseInt(member, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid post id %q in redis: %w", member, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// GetPostOrderScores 批量读取社区内候选帖子的排行分数。
func GetPostOrderScores(ctx context.Context, ids []int64, order string) (map[int64]float64, error) {
	scores := make(map[int64]float64, len(ids))
	if len(ids) == 0 {
		return scores, nil
	}

	key := KeyPostTimeZset
	if order == models.OrderScore {
		key = KeyPostScoreZset
	}
	pipeline := client.Pipeline()
	commands := make(map[int64]*redisClient.FloatCmd, len(ids))
	for _, id := range ids {
		commands[id] = pipeline.ZScore(ctx, getRedisKey(key), strconv.FormatInt(id, 10))
	}
	_, err := pipeline.Exec(ctx)
	if err != nil && !errors.Is(err, redisClient.Nil) {
		return nil, err
	}
	for id, command := range commands {
		score, err := command.Result()
		if errors.Is(err, redisClient.Nil) {
			continue
		}
		if err != nil {
			return nil, err
		}
		scores[id] = score
	}
	return scores, nil
}

// GetPostVoteData 返回每个帖子的净票数（赞成票数减反对票数）。
func GetPostVoteData(ctx context.Context, ids []int64) (map[int64]int64, error) {
	votes := make(map[int64]int64, len(ids))
	if len(ids) == 0 {
		return votes, nil
	}

	type voteCommands struct {
		up   *redisClient.IntCmd
		down *redisClient.IntCmd
	}
	pipeline := client.Pipeline()
	commands := make(map[int64]voteCommands, len(ids))
	for _, id := range ids {
		key := getRedisKey(KeyPostVotedZsetPF + strconv.FormatInt(id, 10))
		commands[id] = voteCommands{
			up:   pipeline.ZCount(ctx, key, "1", "1"),
			down: pipeline.ZCount(ctx, key, "-1", "-1"),
		}
	}
	if _, err := pipeline.Exec(ctx); err != nil {
		return nil, err
	}
	for id, command := range commands {
		votes[id] = command.up.Val() - command.down.Val()
	}
	return votes, nil

}
