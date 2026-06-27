package user

import (
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
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
		UserID:      basicUser.ID,
		FullName:    basicUser.FullName,
		Email:       basicUser.Email,
		Phone:       basicUser.Phone,
		Role:        basicUser.Role,
		IsVerified:  basicUser.IsVerified,
		IsLocked:    basicUser.IsLocked,
		Address:     profile.Address,
		AvatarURL:   profile.AvatarURL,
		Gender:      profile.Gender,
		DateOfBirth: profile.DateOfBirth,
		Age:         calculateAge(profile.DateOfBirth),
	}, nil
}

func (s *Service) UpdateMyProfile(
	userID uint,
	req UpdateUserProfileRequest,
) error {
	profile := &UserProfile{
		UserID:      userID,
		Address:     req.Address,
		AvatarURL:   req.AvatarURL,
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

	return s.repo.UpdateProfile(userID, profile)
}

func calculateAge(dateOfBirth *time.Time) int {
	if dateOfBirth == nil {
		return 0
	}

	now := time.Now()

	age := now.Year() - dateOfBirth.Year()

	if now.YearDay() < dateOfBirth.YearDay() {
		age--
	}

	return age
}
