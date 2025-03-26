package mock

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/mdm/android"
)

func (s *Datastore) InitCommonMocks() {
	s.CreateEnterpriseFunc = func(ctx context.Context, _ uint) (uint, error) {
		return 1, nil
	}
	s.UpdateEnterpriseFunc = func(ctx context.Context, enterprise *android.EnterpriseDetails) error {
		return nil
	}
	s.GetEnterpriseFunc = func(ctx context.Context) (*android.Enterprise, error) {
		return &android.Enterprise{}, nil
	}
	s.GetEnterpriseByIDFunc = func(ctx context.Context, ID uint) (*android.EnterpriseDetails, error) {
		return &android.EnterpriseDetails{}, nil
	}
	s.GetEnterpriseBySignupTokenFunc = func(ctx context.Context, signupToken string) (*android.EnterpriseDetails, error) {
		if signupToken == "signup_token" {
			return &android.EnterpriseDetails{}, nil
		}
		return nil, &notFoundError{errors.New("not found")}
	}
	s.DeleteAllEnterprisesFunc = func(ctx context.Context) error {
		return nil
	}
	s.DeleteOtherEnterprisesFunc = func(ctx context.Context, ID uint) error {
		return nil
	}
}

type notFoundError struct {
	error
}

func (e *notFoundError) IsNotFound() bool {
	return true
}
