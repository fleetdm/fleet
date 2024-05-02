package nvdsync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

func TestVulncheckIndexAPI(t *testing.T) {
	vResp := `
{
    "_benchmark": 0.043638,
    "_meta": {
        "timestamp": "2024-04-29T18:54:33.9958854Z",
        "index": "nist-nvd2",
        "limit": 1,
        "total_documents": 1,
        "sort": "_id",
        "parameters": [
            {
                "name": "cve",
                "format": "CVE-YYYY-N{4-7}"
            },
            {
                "name": "alias"
            },
            {
                "name": "iava",
                "format": "[0-9]{4}[A-Z-0-9]+"
            },
            {
                "name": "threat_actor"
            },
            {
                "name": "mitre_id"
            },
            {
                "name": "misp_id"
            },
            {
                "name": "ransomware"
            },
            {
                "name": "botnet"
            },
            {
                "name": "published"
            },
            {
                "name": "lastModStartDate",
                "format": "2024-04-29"
            },
            {
                "name": "lastModEndDate",
                "format": "YYYY-MM-DD"
            }
        ],
        "order": "desc",
        "next_cursor": "Q1ZFLTIwMjItNDg2NDc="
    },
    "data": [
        {
            "id": "CVE-2024-23205",
            "sourceIdentifier": "product-security@apple.com",
            "vulnStatus": "Awaiting Analysis",
            "published": "2024-03-08T02:15:47.393",
            "lastModified": "2024-03-13T21:15:55.680",
            "descriptions": [
                {
                    "lang": "en",
                    "value": "A privacy issue was addressed with improved private data redaction for log entries. This issue is fixed in macOS Sonoma 14.4, iOS 17.4 and iPadOS 17.4. An app may be able to access sensitive user data."
                },
                {
                    "lang": "es",
                    "value": "Se solucionó un problema de privacidad mejorando la redacción de datos privados para las entradas de registro. Este problema se solucionó en macOS Sonoma 14.4, iOS 17.4 y iPadOS 17.4. Es posible que una aplicación pueda acceder a datos confidenciales del usuario."
                }
            ],
            "references": [
                {
                    "url": "http://seclists.org/fulldisclosure/2024/Mar/21",
                    "source": "product-security@apple.com"
                },
                {
                    "url": "https://support.apple.com/en-us/HT214081",
                    "source": "product-security@apple.com"
                },
                {
                    "url": "https://support.apple.com/en-us/HT214084",
                    "source": "product-security@apple.com"
                }
            ],
            "metrics": {},
            "vcConfigurations": [
                {
                    "nodes": [
                        {
                            "cpeMatch": [
                                {
                                    "vulnerable": true,
                                    "criteria": "cpe:2.3:o:apple:macos:*:*:*:*:*:*:*:*",
                                    "versionEndExcluding": "14.4",
                                    "matchCriteriaId": ""
                                }
                            ]
                        }
                    ]
                },
                {
                    "nodes": [
                        {
                            "cpeMatch": [
                                {
                                    "vulnerable": true,
                                    "criteria": "cpe:2.3:o:apple:iphone_os:*:*:*:*:*:*:*:*",
                                    "versionEndExcluding": "17.4",
                                    "matchCriteriaId": ""
                                }
                            ]
                        }
                    ]
                }
            ],
            "vcVulnerableCPEs": [
                "cpe:2.3:o:apple:macos:1.0:*:*:*:*:*:*:*"
            ],
            "_timestamp": "2024-04-22T01:15:22.872714Z"
        }
    ]
}
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(vResp))
	}))
	defer server.Close()

	s := CVE{
		client:           fleethttp.NewClient(),
		dbDir:            "/tmp",
		logger:           log.NewNopLogger(),
		MaxTryAttempts:   3,
		WaitTimeForRetry: 1 * time.Second,
	}

	resp, err := s.getVulnCheckIndexCVEs(context.Background(), &server.URL, nil, time.Now())
	require.NoError(t, err)

	require.Equal(t, "Q1ZFLTIwMjItNDg2NDc=", resp.Meta.NextCursor)
	require.Len(t, resp.Data, 1)
	require.Equal(t, ptr.String("CVE-2024-23205"), resp.Data[0].ID)
	require.Len(t, resp.Data[0].VcConfigurations, 2)
}
