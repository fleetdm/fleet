package service

import (
	"context"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLicenseService(t *testing.T) {
	tokenString := "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImtpZCI6IjRkOmM1OmRlOmE1Oj" +
		"czOmUxOmE4OjI4OmU2OmEyOjMwOmI4OmI1OjBmOjg4OjQ0In0.eyJsaWNlbnNlX3V1aWQiOiIyZD" +
		"gwMmEyYS1hZjRjLTQ5ZjItYWRlNC0zOGJmNjBmMmQxZjYiLCJvcmdhbml6YXRpb25fbmFtZSI6Il" +
		"BoYW50YXNtLCBJbmMuIiwib3JnYW5pemF0aW9uX3V1aWQiOiI5ZmFiNjdiMy0wZWFjLTRhODMtOTI" +
		"wNS04MjkyMWIwNDJmODYiLCJob3N0X2xpbWl0IjowLCJldmFsdWF0aW9uIjp0cnVlLCJleHBpcmV" +
		"zX2F0IjoiMjAxNy0wMy0wNFQxNDozODo1NyswMDowMCJ9.DRFQIUDFXT0bDdya0IJKvATKCJjv3Mv" +
		"w5gMxHNzby_L80muoe-36DoRxBAJZHL7dOfQDU8NRK2Mt64ozThrhWVl8wJlD9mk5ABe3tNw3LJRl" +
		"2mHvOLmk37_AIHp5AEKZ6cWMPa9zf8hWf6bAv_0rOJf5wgyE81pfqRFtO0OnkGO3WLcP66L0AIntq" +
		"IzAE_vWmizcUvUOCWDqwcBlT-P1mZnWJFCaSBpmpQoi3KEKJDx0wMjLiRNLX9R9dr3v3ojccoYuxR" +
		"qAws-OHv3VzcuGdn3Pt9WBDr4cXdtqxaGtxJb6-BDvp8QQk69ACZXrZJ8NhZAL0EVlviRRw8bbEYchZQ"

	publicKey :=
		`-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0ZhY7r6HmifXPtServt4
D3MSi8Awe9u132vLf8yzlknvnq+8CSnOPSSbCD+HajvZ6dnNJXjdcAhuZ32ShrH8
rEQACEUS8Mh4z8Mo5Nlq1ou0s2JzWCx049kA34jP3u6AiPgpWUf8JRGstTlisxMn
H6B7miDs1038gVbN5rk+j+3ALYzllaTnCX3Y0C7f6IW7BjNO/tvFB84/95xfOLEz
o2MeFMqkD29hvcrUW+8+fQGJaVLvcEqBDnIEVbCCk8Wnoi48dUE06WHUl6voJecD
dW1E6jHcq8PQFK+4bI1gKZVbV4dFGSSMUyD7ov77aWHjxdQe6YEGcSXKzfyMaUtQ
vQIDAQAB
-----END PUBLIC KEY-----
`

	ds := new(mock.Store)
	ds.LicenseFunc = func() (*kolide.License, error) {
		result := &kolide.License{
			UpdateTimestamp: kolide.UpdateTimestamp{
				UpdatedAt: time.Now().Add(-5 * time.Minute),
			},
			Token:     &tokenString,
			PublicKey: publicKey,
			Revoked:   false,
			ID:        1,
		}
		return result, nil
	}

	svc, err := newTestService(ds, nil)
	require.Nil(t, err)
	ctx := context.Background()

	lic, err := svc.License(ctx)
	require.Nil(t, err)
	claims, err := lic.Claims()
	require.Nil(t, err)
	require.NotNil(t, claims)

	assert.Equal(t, "2d802a2a-af4c-49f2-ade4-38bf60f2d1f6", claims.LicenseUUID)
	assert.Equal(t, "Phantasm, Inc.", claims.OrganizationName)
	assert.Equal(t, "9fab67b3-0eac-4a83-9205-82921b042f86", claims.OrganizationUUID)
	assert.Equal(t, 0, claims.HostLimit)
	assert.Equal(t, "2017-03-04T14:38:57Z", claims.ExpiresAt.Format(time.RFC3339))
	assert.True(t, claims.Evaluation)

	// Eval license expires without grace period
	tm, err := time.Parse(time.RFC3339, "2017-03-04T14:38:57Z")
	require.Nil(t, err)
	c := clock.NewMockClock(tm)
	assert.False(t, claims.Expired(c.Now()))
	c.AddTime(time.Second)
	assert.True(t, claims.Expired(c.Now()))
	// Non eval gets a sixty day grace period
	claims.Evaluation = false
	tm, err = time.Parse(time.RFC3339, "2017-03-04T14:38:57Z")
	tm = tm.Add(kolide.LicenseGracePeriod)
	c = clock.NewMockClock(tm)
	assert.False(t, claims.Expired(c.Now()))
	c.AddTime(time.Second)
	assert.True(t, claims.Expired(c.Now()))

}

func TestLicenseServiceWithTamperedToken(t *testing.T) {
	tokenString := "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImtpZCI6IjRkOmM1OmRlOmE1Oj" +
		"czOmUxOmE4OjI4OmU2OmEyOjMwOmI4OmI1OjBmOjg4OjQ0In0.eyJsaWNlbnNlX3V1aWQiOiIyZD" +
		"gwMmEyYS1hZjRjLTQ5ZjItYWRlNC0zOGJmNjBmMmQxZjYiLCJvcmdhbml6YXRpb25fbmFtZSI6Il" +
		"BoYW50YXNtLCBJbmMuIiwib35nYW5pemF0aW9uX3V1aWQiOiI5ZmFiNjdiMy0wZWFjLTRhODMtOTI" +
		"wNS04MjkyMWIwNDJmODYiLCJob3N0X2xpbWl0IjowLCJldmFsdWF0aW9uIjp0cnVlLCJleHBpcmV" +
		"zX2F0IjoiMjAxNy0wMy0wNFQxNDozODo1NyswMDowMCJ9.DRFQIUDFXT0bDdya0IJKvATKCJjv3Mv" +
		"w5gMxHNzby_L80muoe-36DoRxBAJZHL7dOfQDU8NRK2Mt64ozThrhWVl8wJlD9mk5ABe3tNw3LJRl" +
		"2mHvOLmk37_AIHp5AEKZ6cWMPa9zf8hWf6bAv_0rOJf5wgyE81pfqRFtO0OnkGO3WLcP66L0AIntq" +
		"IzAE_vWmizcUvUOCWDqwcBlT-P1mZnWJFCaSBpmpQoi3KEKJDx0wMjLiRNLX9R9dr3v3ojccoYuxR" +
		"qAws-OHv3VzcuGdn3Pt9WBDr4cXdtqxaGtxJb6-BDvp8QQk69ACZXrZJ8NhZAL0EVlviRRw8bbEYchZQ"

	publicKey :=
		`-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0ZhY7r6HmifXPtServt4
D3MSi8Awe9u132vLf8yzlknvnq+8CSnOPSSbCD+HajvZ6dnNJXjdcAhuZ32ShrH8
rEQACEUS8Mh4z8Mo5Nlq1ou0s2JzWCx049kA34jP3u6AiPgpWUf8JRGstTlisxMn
H6B7miDs1038gVbN5rk+j+3ALYzllaTnCX3Y0C7f6IW7BjNO/tvFB84/95xfOLEz
o2MeFMqkD29hvcrUW+8+fQGJaVLvcEqBDnIEVbCCk8Wnoi48dUE06WHUl6voJecD
dW1E6jHcq8PQFK+4bI1gKZVbV4dFGSSMUyD7ov77aWHjxdQe6YEGcSXKzfyMaUtQ
vQIDAQAB
-----END PUBLIC KEY-----
`

	ds := new(mock.Store)
	ds.LicenseFunc = func() (*kolide.License, error) {
		result := &kolide.License{
			UpdateTimestamp: kolide.UpdateTimestamp{
				UpdatedAt: time.Now().Add(-5 * time.Minute),
			},
			Token:     &tokenString,
			PublicKey: publicKey,
			Revoked:   false,
			ID:        1,
		}
		return result, nil
	}

	svc, err := newTestService(ds, nil)
	require.Nil(t, err)
	ctx := context.Background()

	lic, err := svc.License(ctx)
	require.Nil(t, err)
	_, err = lic.Claims()
	require.NotNil(t, err)

}

func TestLicenseServiceWithMissingToken(t *testing.T) {

	publicKey :=
		`-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0ZhY7r6HmifXPtServt4
D3MSi8Awe9u132vLf8yzlknvnq+8CSnOPSSbCD+HajvZ6dnNJXjdcAhuZ32ShrH8
rEQACEUS8Mh4z8Mo5Nlq1ou0s2JzWCx049kA34jP3u6AiPgpWUf8JRGstTlisxMn
H6B7miDs1038gVbN5rk+j+3ALYzllaTnCX3Y0C7f6IW7BjNO/tvFB84/95xfOLEz
o2MeFMqkD29hvcrUW+8+fQGJaVLvcEqBDnIEVbCCk8Wnoi48dUE06WHUl6voJecD
dW1E6jHcq8PQFK+4bI1gKZVbV4dFGSSMUyD7ov77aWHjxdQe6YEGcSXKzfyMaUtQ
vQIDAQAB
-----END PUBLIC KEY-----
`

	ds := new(mock.Store)
	ds.LicenseFunc = func() (*kolide.License, error) {
		result := &kolide.License{
			UpdateTimestamp: kolide.UpdateTimestamp{
				UpdatedAt: time.Now().Add(-5 * time.Minute),
			},
			Token:     nil,
			PublicKey: publicKey,
			Revoked:   false,
			ID:        1,
		}
		return result, nil
	}

	svc, err := newTestService(ds, nil)
	require.Nil(t, err)
	ctx := context.Background()

	lic, err := svc.License(ctx)
	require.Nil(t, err)
	_, err = lic.Claims()
	require.NotNil(t, err)

}

func TestLicenseServiceWithWrongPublicKey(t *testing.T) {
	tokenString := "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImtpZCI6IjRkOmM1OmRlOmE1Oj" +
		"czOmUxOmE4OjI4OmU2OmEyOjMwOmI4OmI1OjBmOjg4OjQ0In0.eyJsaWNlbnNlX3V1aWQiOiIyZD" +
		"gwMmEyYS1hZjRjLTQ5ZjItYWRlNC0zOGJmNjBmMmQxZjYiLCJvcmdhbml6YXRpb25fbmFtZSI6Il" +
		"BoYW50YXNtLCBJbmMuIiwib3JnYW5pemF0aW9uX3V1aWQiOiI5ZmFiNjdiMy0wZWFjLTRhODMtOTI" +
		"wNS04MjkyMWIwNDJmODYiLCJob3N0X2xpbWl0IjowLCJldmFsdWF0aW9uIjp0cnVlLCJleHBpcmV" +
		"zX2F0IjoiMjAxNy0wMy0wNFQxNDozODo1NyswMDowMCJ9.DRFQIUDFXT0bDdya0IJKvATKCJjv3Mv" +
		"w5gMxHNzby_L80muoe-36DoRxBAJZHL7dOfQDU8NRK2Mt64ozThrhWVl8wJlD9mk5ABe3tNw3LJRl" +
		"2mHvOLmk37_AIHp5AEKZ6cWMPa9zf8hWf6bAv_0rOJf5wgyE81pfqRFtO0OnkGO3WLcP66L0AIntq" +
		"IzAE_vWmizcUvUOCWDqwcBlT-P1mZnWJFCaSBpmpQoi3KEKJDx0wMjLiRNLX9R9dr3v3ojccoYuxR" +
		"qAws-OHv3VzcuGdn3Pt9WBDr4cXdtqxaGtxJb6-BDvp8QQk69ACZXrZJ8NhZAL0EVlviRRw8bbEYchZQ"

	publicKey :=
		`-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiBBB0BAQEFAAOCAQ8AMIIBCgKCAQEA0ZhY7r6HmifXPtServt4
D3MSi8Awe9u132vLf8yzlknvnq+8CSnOPSSbCD+HajvZ6dnNJXjdcAhuZ32ShrH8
rEQACEUS8Mh4z8Mo5Nlq1ou0s2JzWCx049kA34jP3u6AiPgpWUf8JRGstTlisxMn
H6B7miDs1038gVbN5rk+j+3ALYzllaTnCX3Y0C7f6IW7BjNO/tvFB84/95xfOLEz
o2MeFMqkD29hvcrUW+8+fQGJaVLvcEqBDnIEVbCCk8Wnoi48dUE06WHUl6voJecD
dW1E6jHcq8PQFK+4bI1gKZVbV4dFGSSMUyD7ov77aWHjxdQe6YEGcSXKzfyMaUtQ
vQIDAQAB
-----END PUBLIC KEY-----
`

	ds := new(mock.Store)
	ds.LicenseFunc = func() (*kolide.License, error) {
		result := &kolide.License{
			UpdateTimestamp: kolide.UpdateTimestamp{
				UpdatedAt: time.Now().Add(-5 * time.Minute),
			},
			Token:     &tokenString,
			PublicKey: publicKey,
			Revoked:   false,
			ID:        1,
		}
		return result, nil
	}

	svc, err := newTestService(ds, nil)
	require.Nil(t, err)
	ctx := context.Background()

	lic, err := svc.License(ctx)
	require.Nil(t, err)
	_, err = lic.Claims()
	require.NotNil(t, err)

}
