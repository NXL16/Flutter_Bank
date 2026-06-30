package user

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetMyProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	profile, err := h.service.GetMyProfile(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Lấy thông tin profile thành công",
		"data":    profile,
	})
}

func (h *Handler) UpdateMyProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req UpdateUserProfileRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Dữ liệu không hợp lệ",
			"error":   err.Error(),
		})
		return
	}

	if err := h.service.UpdateMyProfile(userID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cập nhật profile thành công",
	})
}

func (h *Handler) UploadMyAvatar(c *gin.Context) {
	const maxAvatarSize = int64(8 << 20)
	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		maxAvatarSize+(1<<20),
	)

	header, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Vui lòng chọn một ảnh đại diện",
		})
		return
	}
	if header.Size <= 0 || header.Size > maxAvatarSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"success": false,
			"message": "Ảnh đại diện không được vượt quá 8 MB",
		})
		return
	}

	file, err := header.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Không thể đọc ảnh đại diện",
		})
		return
	}
	defer file.Close()

	sample := make([]byte, 512)
	readBytes, readErr := io.ReadFull(file, sample)
	if readErr != nil && readErr != io.ErrUnexpectedEOF {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Ảnh đại diện không hợp lệ",
		})
		return
	}
	contentType := http.DetectContentType(sample[:readBytes])
	allowedType := contentType == "image/jpeg" ||
		contentType == "image/png" ||
		contentType == "image/webp"
	if !allowedType {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{
			"success": false,
			"message": "Chỉ hỗ trợ ảnh JPG, PNG hoặc WEBP",
		})
		return
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Không thể xử lý ảnh đại diện",
		})
		return
	}

	avatarURL, err := h.service.UploadMyAvatar(
		c.Request.Context(),
		c.GetUint("user_id"),
		file,
		header.Filename,
		contentType,
	)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cập nhật ảnh đại diện thành công",
		"data":    gin.H{"avatar_url": avatarURL},
	})
}
