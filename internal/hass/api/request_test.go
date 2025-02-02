// Copyright (c) 2024 Joshua Rich <joshua.rich@gmail.com>
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/joshuar/go-hass-agent/internal/preferences"
)

var defaultTestPrefs = []preferences.Preference{
	preferences.Token("testToken"),
	preferences.CloudhookURL(""),
	preferences.RemoteUIURL(""),
	preferences.WebhookID("testID"),
	preferences.Secret(""),
	preferences.DeviceName("testDevice"),
	preferences.DeviceID("testID"),
	preferences.Version("6.4.0"),
	preferences.Registered(true),
}

func mockServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		req := &UnencryptedRequest{}
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.Nil(t, err)
		switch req.Type {
		case "register_sensor":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true}`))
		case "encrypted":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true}`))
		}
	}))
}

func Test_marshalJSON(t *testing.T) {
	mockReq := &RequestMock{
		RequestDataFunc: func() json.RawMessage {
			return json.RawMessage(`{"someField": "someValue"}`)
		},
		RequestTypeFunc: func() RequestType {
			return RequestTypeUpdateSensorStates
		},
	}
	mockEncReq := &RequestMock{
		RequestDataFunc: func() json.RawMessage {
			return json.RawMessage(`{"someField": "someValue"}`)
		},
		RequestTypeFunc: func() RequestType {
			return RequestTypeEncrypted
		},
	}

	type args struct {
		request Request
		secret  string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "unencrypted request",
			args: args{request: mockReq},
			want: []byte(`{"type":"update_sensor_states","data":{"someField":"someValue"}}`),
		},
		{
			name:    "encrypted request without secret",
			args:    args{request: mockEncReq},
			want:    nil,
			wantErr: true,
		},
		{
			name: "encrypted request with secret",
			args: args{request: mockEncReq, secret: "fakeSecret"},
			want: []byte(`{"type":"encrypted","encrypted_data":{"someField":"someValue"},"encrypted":true}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := marshalJSON(tt.args.request, tt.args.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("marshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("marshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecuteRequest(t *testing.T) {
	mockServer := mockServer(t)
	defer mockServer.Close()

	preferences.SetPath(t.TempDir())
	prefs := defaultTestPrefs
	prefs = append(prefs,
		preferences.Host(mockServer.URL),
		preferences.RestAPIURL(mockServer.URL),
		preferences.WebsocketURL(mockServer.URL),
	)
	err := preferences.Save(prefs...)
	assert.Nil(t, err)
	p, err := preferences.Load()
	assert.Nil(t, err)
	ctx := preferences.EmbedInContext(context.TODO(), p)
	mockReq := &RequestMock{
		RequestDataFunc: func() json.RawMessage {
			return json.RawMessage(`{"someField": "someValue"}`)
		},
		RequestTypeFunc: func() RequestType {
			return RequestTypeUpdateSensorStates
		},
	}
	type args struct {
		ctx     context.Context
		request Request
	}
	tests := []struct {
		name string
		args args
		want chan any
	}{
		{
			name: "default test",
			args: args{ctx: ctx, request: mockReq},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ExecuteRequest(tt.args.ctx, tt.args.request)
			// if got := ExecuteRequest(tt.args.ctx, tt.args.request); !reflect.DeepEqual(got, tt.want) {
			// 	t.Errorf("ExecuteRequest() = %v, want %v", got, tt.want)
			// }
		})
	}
}
