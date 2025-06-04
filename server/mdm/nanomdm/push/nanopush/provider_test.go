package nanopush

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/stretchr/testify/require"
)

func TestPush(t *testing.T) {
	// our "raw" push info
	devicePushInfoStrings := [][]string{
		{
			"c2732227a1d8021cfaf781d71fb2f908c61f5861079a00954a5453f1d0281433",
			"47250C9C-1B37-4381-98A9-0B8315A441C7",
			"com.example.apns-topic",
		},
	}

	// test a single push
	t.Run("single-push", func(t *testing.T) {
		testPushDevices(t, devicePushInfoStrings)
	})

	devicePushInfoStrings = append(devicePushInfoStrings, []string{
		"7f1839ca30d5c6d36d6ae426258c4306c14eca90afd709a07375a85ad5a11c69",
		"1C0B33FD-9336-4A7A-A080-7BEA9BD032EC",
		"com.example.apns-topic",
	})

	// test a multiple push
	t.Run("multiple-push", func(t *testing.T) {
		testPushDevices(t, devicePushInfoStrings)
	})
}

func testPushDevices(t *testing.T, input [][]string) {
	// assemble it into a list and map
	devices := make(map[string]*mdm.Push)
	var pushInfos []*mdm.Push
	for _, devicePushInfos := range input {
		pushInfo := &mdm.Push{
			PushMagic: devicePushInfos[1],
			Topic:     devicePushInfos[2],
		}
		err := pushInfo.SetTokenString(devicePushInfos[0])
		require.NoError(t, err)
		devices[devicePushInfos[0]] = pushInfo
		pushInfos = append(pushInfos, pushInfo)
	}

	apnsID := "922D9F1F-B82E-B337-EDC9-DB4FC8527676"

	handler := http.NewServeMux()

	handler.HandleFunc("/3/device/", func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.String()
		var device string
		var pushMagic string
		if len(url) > 11 && url[:10] == "/3/device/" {
			device = url[10:]
			if _, ok := devices[device]; !ok {
				t.Errorf("device id not present: %s", device)
			} else {
				pushMagic = devices[device].PushMagic
			}
		} else {
			t.Fatal("invalid URL form")
		}

		payload := []byte(`{"mdm":"` + pushMagic + `"}`)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if have, want := body, payload; !bytes.Equal(have, want) {
			t.Errorf("body: have %q, want %q", string(have), string(want))
		}

		w.Header().Set("apns-id", apnsID)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	prov := &Provider{
		baseURL: server.URL,
		client:  http.DefaultClient,
	}

	resp, err := prov.Push(context.Background(), pushInfos)
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range resp {
		if _, ok := devices[k]; !ok || v == nil {
			t.Errorf("device not found (or is nil): %s", k)
		} else {
			if have, want := v.Id, apnsID; have != want {
				t.Errorf("url: have %q, want %q", have, want)
			}
		}
	}

}
