package favorite

import (
	"TikTokk/api"
	"TikTokk/internal/TikTokk/model"
	"TikTokk/internal/TikTokk/store"
	"TikTokk/tools"
	"context"
	"strconv"

	"gorm.io/gorm"
)

type FavoriteRelationBiz interface {
	Action(ctx context.Context, videoID, userID, ActionType uint) error
	List(ctx context.Context, userID uint) (*api.FavoriteListRsp, error)
}

type BFavoriteRelation struct {
	ds store.DataStore
}

var _ FavoriteRelationBiz = (*BFavoriteRelation)(nil)

func New(db store.DataStore) *BFavoriteRelation {
	return &BFavoriteRelation{ds: db}
}

func (b *BFavoriteRelation) List(ctx context.Context, userID uint) (*api.FavoriteListRsp, error) {
	var rsp api.FavoriteListRsp
	//根据用户id查询喜欢列表
	//查询用户喜欢视频数
	u, err := b.ds.Users().Get(ctx, &model.User{UserID: userID})
	if err != nil {
		return &rsp, err
	}
	//得到所有喜欢的视频
	l, err := b.ds.VideoFavoriteRelation().ListLen(ctx, userID, u.FavoriteCount)
	if err != nil {
		return &rsp, err
	}
	//将喜欢的视频转化
	rspList := make([]api.VideoDetailRespond, u.FavoriteCount)
	for i, v := range l {
		//获取作者信息
		author, err := b.ds.Users().Get(ctx, &model.User{UserID: v.AuthorId})
		rspList[i] = *tools.VideoToRsp(&v, author)
		//喜欢的视频状态为喜欢
		rspList[i].IsFavorite = true
		//获取视频作者关注关系
		rel, err := b.ds.UserFollowRelation().FirstOrCreate(ctx, u.UserID, author.UserID, u.Name, author.Name)
		if err != nil {
			return &rsp, err
		}
		rspList[i].Author.IsFollow = rel.IsFollow
	}
	rsp.VideoList = rspList
	return &rsp, nil
}

func (b *BFavoriteRelation) Action(ctx context.Context, videoID, userID, ActionType uint) error {
	//判断点赞关系中点赞状态和操作类型代表的点赞状态是否相同，不相同则修改，相同则不修改
	//获取点赞类型
	var op int
	if ActionType == 1 {
		op = 1
	} else {
		op = -1
	}
	//用户与视频的点赞关系
	user, err := b.ds.Users().Get(ctx, &model.User{UserID: userID})
	if err != nil {
		return err
	}
	rel, err := b.ds.VideoFavoriteRelation().FirstOrCreate(ctx, videoID, user.UserID, user.Name)
	if err != nil {
		return err
	}
	//如果跟原来的点赞状态不同,则进行修改
	if !rel.ISFavorite && op == 1 || rel.ISFavorite && op == -1 {
		//获取作者、视频信息
		video, err := b.ds.Videos().Get(ctx, &model.Video{VideoID: videoID})
		if err != nil {
			return err
		}
		author, err := b.ds.Users().Get(ctx, &model.User{UserID: video.AuthorId})
		if err != nil {
			return err
		}
		authorTotalF, err := strconv.Atoi(author.TotalFavorite)
		if err != nil {
			return err
		}

		//事务
		f := func(tx *gorm.DB) error {
			//修改点赞关系
			if err := tx.Model(&model.UserFavorite{}).
				Where("user_id=? AND user_name=? AND video_id =?", user.UserID, user.Name, video.VideoID).
				Update("is_favorite", !rel.ISFavorite).Error; err != nil {
				return err
			}
			//用户喜欢数+1/-1
			if err := tx.Model(&user).Update("favorite_count", int(user.FavoriteCount)+op).Error; err != nil {
				return err
			}
			//作者获赞数+1/-1
			if err := tx.Model(&author).Update("total_favorite", authorTotalF+op).Error; err != nil {
				return err
			}
			//视频点赞数+1/-1
			if err := tx.Model(&video).Update("favorite_count", int(video.FavoriteCount)+op).Error; err != nil {
				return err
			}
			return nil
		}

		if err := b.ds.VideoFavoriteRelation().Transaction(ctx, f); err != nil {
			return err
		}

	}

	return nil
}
