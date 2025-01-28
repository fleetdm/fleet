package mdm

import (
	"errors"
	"fmt"
)

// Shared iPad users have a static UserID that they connect to MDM with.
// In this case the MDM spec says to fallback to the UserShortName
// which should contain the managed AppleID.
const SharediPadUserID = "FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF"

// EnrollType identifies the type of enrollment.
type EnrollType uint

const (
	Device = 1 + iota
	User
	UserEnrollmentDevice
	UserEnrollment
	SharediPad
	maxEnrollType
)

// Valid tests the validity of the enrollment type
func (et EnrollType) Valid() bool {
	return et > 0 && et < maxEnrollType
}

func (et EnrollType) String() string {
	switch et {
	case Device:
		return "Device"
	case User:
		return "User"
	case UserEnrollmentDevice:
		return "User Enrollment (Device)"
	case UserEnrollment:
		return "User Enrollment"
	case SharediPad:
		return "Shared iPad"
	default:
		return "unknown enroll type value " + fmt.Sprint(uint(et))
	}
}

// ResolvedEnrollment is a sort of collapsed form of Enrollment.
type ResolvedEnrollment struct {
	Type            EnrollType
	DeviceChannelID string
	UserChannelID   string
	IsUserChannel   bool
}

func (resolved *ResolvedEnrollment) Validate() error {
	if resolved == nil {
		return errors.New("nil resolved enrollment")
	}
	if resolved.DeviceChannelID == "" {
		return errors.New("empty device channel id")
	}
	if !resolved.Type.Valid() {
		return errors.New("invalid resolved type")
	}
	return nil
}

// Resolved assembles a ResolvedEnrollment from an Enrollment
func (e *Enrollment) Resolved() (r *ResolvedEnrollment) {
	if e == nil {
		return
	}
	if e.UDID != "" {
		r = new(ResolvedEnrollment)
		r.Type = Device
		r.DeviceChannelID = e.UDID
		if e.UserID != "" {
			r.IsUserChannel = true
			if e.UserID == SharediPadUserID {
				r.Type = SharediPad
				r.UserChannelID = e.UserShortName
			} else {
				r.Type = User
				r.UserChannelID = e.UserID
			}
		}
	} else if e.EnrollmentID != "" {
		r = new(ResolvedEnrollment)
		r.Type = UserEnrollmentDevice
		r.DeviceChannelID = e.EnrollmentID
		if e.EnrollmentUserID != "" {
			r.IsUserChannel = true
			r.Type = UserEnrollment
			r.UserChannelID = e.EnrollmentUserID
		}
	}
	return
}
