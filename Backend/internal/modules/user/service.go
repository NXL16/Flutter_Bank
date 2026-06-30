package user

import (
	"context"
	"io"
)

type AvatarUploader interface {
	UploadAvatar(
		ctx context.Context,
		userID uint,
		file io.Reader,
		filename string,
		contentType string,
	) (string, error)
}

type Service struct {
	repo           *Repository
	avatarUploader AvatarUploader
}

func NewService(repo *Repository, avatarUploader AvatarUploader) *Service {
	return &Service{
		repo:           repo,
		avatarUploader: avatarUploader,
	}
}

func (s *Service) CreateEmptyProfile(userID uint) error {
	existingProfile, err := s.repo.FindByUserID(userID)
	if err != nil {
		return err
	}

	if existingProfile != nil {
		return nil
	}

	profile := &UserProfile{
		UserID: userID,
	}

	return s.repo.CreateProfile(profile)
}

func (s *Service) GetMyProfile(userID uint) (*UserProfileResponse, error) {
	basicUser, err := s.repo.FindBasicUserInfo(userID)
	if err != nil {
		return nil, err
	}

	profile, err := s.repo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	if profile == nil {
		profile = &UserProfile{
			UserID: userID,
		}
	}

	return &UserProfileResponse{
		FullName:    basicUser.FullName,
		Phone:       basicUser.Phone,
		Role:        basicUser.Role,
		Address:     profile.Address,
		AvatarURL:   profile.AvatarURL,
		Gender:      profile.Gender,
		DateOfBirth: profile.DateOfBirth,
	}, nil
}

func (s *Service) UpdateMyProfile(
	userID uint,
	req UpdateUserProfileRequest,
) error {
	profile := &UserProfile{
		UserID:      userID,
		Address:     req.Address,
		Gender:      req.Gender,
		DateOfBirth: req.DateOfBirth,
	}

	existingProfile, err := s.repo.FindByUserID(userID)
	if err != nil {
		return err
	}

	if existingProfile == nil {
		return s.repo.CreateProfile(profile)
	}

	profile.AvatarURL = existingProfile.AvatarURL
	return s.repo.UpdateProfile(userID, profile)
}

func (s *Service) UploadMyAvatar(
	ctx context.Context,
	userID uint,
	file io.Reader,
	filename string,
	contentType string,
) (string, error) {
	avatarURL, err := s.avatarUploader.UploadAvatar(
		ctx,
		userID,
		file,
		filename,
		contentType,
	)
	if err != nil {
		return "", err
	}

	existingProfile, err := s.repo.FindByUserID(userID)
	if err != nil {
		return "", err
	}
	if existingProfile == nil {
		err = s.repo.CreateProfile(&UserProfile{
			UserID:    userID,
			AvatarURL: avatarURL,
		})
	} else {
		err = s.repo.UpdateAvatarURL(userID, avatarURL)
	}
	if err != nil {
		return "", err
	}
	return avatarURL, nil
}
