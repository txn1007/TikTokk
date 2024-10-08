package comment

import (
	"TikTokk/api"
	"TikTokk/internal/TikTokk/model"
	"TikTokk/internal/TikTokk/store"
	"TikTokk/tools"
	"context"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type CommentBiz interface {
	Delete(ctx context.Context, commentID, videoID, userID uint) error
	Create(ctx context.Context, videoID, userID uint, commentText string) (*api.CommentActionRsp, error)
	List(ctx context.Context, videoID uint) (*api.CommentListRsp, error)
}

type BComment struct {
	ds store.DataStore
}

var _ CommentBiz = (*BComment)(nil)

func New(s store.DataStore) *BComment {
	return &BComment{ds: s}
}

func (b BComment) List(ctx context.Context, videoID uint) (*api.CommentListRsp, error) {
	var rsp api.CommentListRsp
	//store 获取所有评论,按倒序
	list, err := b.ds.Comment().List(ctx, videoID)
	if err != nil {
		return &rsp, err
	}
	//返回
	rList := make([]api.CommentDetailRsp, len(list))
	for i, v := range list {
		rList[i] = *tools.CommentToRsp(&v)
		u, err := b.ds.Users().Get(ctx, &model.User{UserID: v.UserId})
		if err != nil {
			return &rsp, err
		}
		rList[i].User = *tools.UserToRsp(u)
	}
	rsp.CommentList = rList
	return &rsp, nil
}

func (b BComment) Delete(ctx context.Context, commentID, videoID, userID uint) error {
	//得到comment具体信息,对比申请删除是否同作者相同
	comment, err := b.ds.Comment().Get(ctx, &model.Comment{CommentID: commentID})
	if err != nil {
		return err
	}
	if comment.UserId != userID {
		return fmt.Errorf("非评论发布者")
	}
	//获取视频信息得到评论数
	v, err := b.ds.Videos().Get(ctx, &model.Video{VideoID: videoID})
	if err != nil {
		return err
	}
	//创建事务,删除记录,视频评论数-1
	f := func(tx *gorm.DB) error {
		if err := tx.Table("comments").Delete(&model.Comment{CommentID: commentID}).Error; err != nil {
			return err
		}
		if err := tx.Table("videos").Model(&model.Video{VideoID: videoID}).Update(
			"comment_count", v.CommentCount-1).Error; err != nil {
			return err
		}
		return nil
	}
	if err := b.ds.Comment().Transaction(ctx, f); err != nil {
		return err
	}
	return nil
}

func (b BComment) Create(ctx context.Context, videoID, userID uint, commentText string) (*api.CommentActionRsp, error) {
	var rsp api.CommentActionRsp
	//得到视频总评论数
	v, err := b.ds.Videos().Get(ctx, &model.Video{VideoID: videoID})
	if err != nil {
		return &rsp, err
	}
	//得到作者信息,并转化
	u, err := b.ds.Users().Get(ctx, &model.User{UserID: userID})
	if err != nil {
		return &rsp, err
	}
	rU := tools.UserToRsp(u)
	//创建者同登录账号相同,故未关注
	rU.IsFollow = false
	//创建时间为服务器当前时间
	nowTime := time.Now().Format("01-02")
	//创建结构体
	c := model.Comment{VideoId: videoID, CreateDate: nowTime, Content: commentText, UserName: u.Name, UserId: u.UserID}
	//创建事务,创建记录,视频评论数+1
	f := func(tx *gorm.DB) error {
		if err := tx.Table("comments").Create(&c).Error; err != nil {
			return err
		}
		if err := tx.Table("videos").Model(&model.Video{VideoID: videoID}).Update(
			"comment_count", v.CommentCount+1).Error; err != nil {
			return err
		}
		return nil
	}
	if err := b.ds.Comment().Transaction(ctx, f); err != nil {
		return &rsp, err
	}
	//得到创建完成的信息
	com, err := b.ds.Comment().Get(ctx, &model.Comment{VideoId: videoID, UserId: userID, CreateDate: nowTime})
	if err != nil {
		return &rsp, err
	}
	rsp.Comment = *tools.CommentToRsp(com)
	rsp.Comment.User = *rU
	return &rsp, err

}
