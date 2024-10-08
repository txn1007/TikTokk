package controller

import (
	"TikTokk/internal/pkg/token"
	"errors"
	"github.com/gin-gonic/gin"
	"log"
	"strconv"
)

func GetUserID(ctx *gin.Context) (uint, error) {
	userIDStr := ctx.GetString(token.Config.IdentityKey)
	log.Println("userIDStr=", userIDStr)
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 0, err
	}
	if userID < 0 {
		return 0, errors.New("invalid id")
	}
	return uint(userID), nil
}
