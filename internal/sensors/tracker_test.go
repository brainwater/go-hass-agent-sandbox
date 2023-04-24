// Copyright (c) 2023 Joshua Rich <joshua.rich@gmail.com>
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package sensors

import (
	"context"
	"os"
	"reflect"
	"sync"
	"testing"

	"fyne.io/fyne/v2/app"
	"github.com/joshuar/go-hass-agent/internal/device"
	"github.com/joshuar/go-hass-agent/internal/hass"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockSensorUpdate struct {
	mock.Mock
	id         string
	state      interface{}
	icon       string
	attributes interface{}
}

func (m *mockSensorUpdate) Name() string {
	m.On("Name")
	args := m.Called()
	return args.String()
}

func (m *mockSensorUpdate) ID() string {
	m.On("ID")
	args := m.Called()
	if m.id == "" {
		return args.String()
	} else {
		return m.id
	}
}

func (m *mockSensorUpdate) Icon() string {
	m.On("Icon")
	args := m.Called()
	if m.icon == "" {
		return args.String()
	} else {
		return m.icon
	}
}

func (m *mockSensorUpdate) SensorType() hass.SensorType {
	m.On("SensorType")
	m.Called()
	return hass.TypeSensor
}

func (m *mockSensorUpdate) DeviceClass() hass.SensorDeviceClass {
	m.On("DeviceClass")
	m.Called()
	return 0
}

func (m *mockSensorUpdate) StateClass() hass.SensorStateClass {
	m.On("StateClass")
	m.Called()
	return 0
}

func (m *mockSensorUpdate) State() interface{} {
	m.On("State")
	args := m.Called()
	if m.state == nil {
		return args.String()
	} else {
		return m.state
	}

}

func (m *mockSensorUpdate) Units() string {
	m.On("Units")
	args := m.Called()
	return args.String()
}

func (m *mockSensorUpdate) Category() string {
	m.On("Category")
	args := m.Called()
	return args.String()
}

func (m *mockSensorUpdate) Attributes() interface{} {
	m.On("Attributes")
	m.Called()
	return m.attributes
}

type MockSensorRegistry struct {
	mock.Mock
}

func newMockSensorTracker(t *testing.T) *sensorTracker {
	fakeRegistry := newMockSensorRegistry(t)
	fakeTracker := &sensorTracker{
		sensor:        make(map[string]*sensorState),
		sensorWorkers: nil,
		registry:      fakeRegistry,
		hassConfig:    nil,
	}
	return fakeTracker
}

var testApp = app.NewWithID("org.joshuar.go-hass-agent-test")
var uri = testApp.Storage().RootURI()

func TestNewSensorTracker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracker := NewSensorTracker(ctx, uri)
	assert.IsType(t, &sensorTracker{}, tracker)

	os.RemoveAll(uri.Path())
}

func Test_sensorTracker_add(t *testing.T) {
	type fields struct {
		mu            sync.RWMutex
		sensor        map[string]*sensorState
		sensorWorkers *device.SensorInfo
		registry      *sensorRegistry
		hassConfig    *hass.HassConfig
	}
	type args struct {
		s hass.SensorUpdate
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "successful add",
			fields: fields{
				registry: newMockSensorRegistry(t),
				sensor:   make(map[string]*sensorState)},
			args: args{s: &mockSensorUpdate{}},
		},
		{
			name: "unsuccessful add",
			fields: fields{
				registry: newMockSensorRegistry(t)},
			args:    args{s: &mockSensorUpdate{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := &sensorTracker{
				mu:            tt.fields.mu,
				sensor:        tt.fields.sensor,
				sensorWorkers: tt.fields.sensorWorkers,
				registry:      tt.fields.registry,
				hassConfig:    tt.fields.hassConfig,
			}
			if err := tracker.add(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("sensorTracker.add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_sensorTracker_Update(t *testing.T) {
	type fields struct {
		mu            sync.RWMutex
		sensor        map[string]*sensorState
		sensorWorkers *device.SensorInfo
		registry      *sensorRegistry
		hassConfig    *hass.HassConfig
	}
	type args struct {
		ctx context.Context
		s   hass.SensorUpdate
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := &sensorTracker{
				mu:            tt.fields.mu,
				sensor:        tt.fields.sensor,
				sensorWorkers: tt.fields.sensorWorkers,
				registry:      tt.fields.registry,
				hassConfig:    tt.fields.hassConfig,
			}
			tracker.Update(tt.args.ctx, tt.args.s)
		})
	}
}

func Test_sensorTracker_get(t *testing.T) {
	fakeSensorUpdate := &mockSensorUpdate{}
	tracker := newMockSensorTracker(t)
	tracker.add(fakeSensorUpdate)
	fakeSensorState := tracker.get(fakeSensorUpdate.ID())
	type args struct {
		id string
	}
	tests := []struct {
		name string
		args args
		want *sensorState
	}{
		{
			name: "existing sensor",
			args: args{id: fakeSensorUpdate.ID()},
			want: fakeSensorState,
		},
		{
			name: "nonexisting sensor",
			args: args{id: "nonexistent"},
			want: nil,
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tracker.get(tt.args.id); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sensorTracker.get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sensorTracker_update(t *testing.T) {
	fakeSensorUpdate := &mockSensorUpdate{}
	fakeSensorStates := make(map[string]*sensorState)
	fakeSensorStates[fakeSensorUpdate.ID()] = marshalSensorState(fakeSensorUpdate)
	type fields struct {
		mu            sync.RWMutex
		sensor        map[string]*sensorState
		sensorWorkers *device.SensorInfo
		registry      *sensorRegistry
		hassConfig    *hass.HassConfig
	}
	type args struct {
		s hass.SensorUpdate
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "try to update nonexistent sensor",
			fields: fields{sensor: make(map[string]*sensorState)},
			args: args{s: &mockSensorUpdate{
				state: "foo",
				icon:  "bar",
			}},
		},
		{
			name:   "try to update existing sensor",
			fields: fields{sensor: fakeSensorStates},
			args: args{s: &mockSensorUpdate{
				state: "foo",
				icon:  "bar",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := &sensorTracker{
				mu:            tt.fields.mu,
				sensor:        tt.fields.sensor,
				sensorWorkers: tt.fields.sensorWorkers,
				registry:      tt.fields.registry,
				hassConfig:    tt.fields.hassConfig,
			}
			tracker.update(tt.args.s)
		})
	}
}

func Test_sensorTracker_exists(t *testing.T) {
	fakeSensorUpdate := &mockSensorUpdate{}
	tracker := newMockSensorTracker(t)
	tracker.add(fakeSensorUpdate)

	type args struct {
		id string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nonexisting sensor",
			args: args{id: "nonexisting"},
			want: false,
		},
		{
			name: "existing sensor",
			args: args{id: fakeSensorUpdate.ID()},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tracker.exists(tt.args.id); got != tt.want {
				t.Errorf("sensorTracker.exists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sensorTracker_StartWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updateCh := make(chan interface{})
	defer close(updateCh)

	fakeWorkerFunc := func(context.Context, chan interface{}) {}

	fakeWorkers := device.NewSensorInfo()
	fakeWorkers.Add("fakeSensor", fakeWorkerFunc)

	type fields struct {
		mu            sync.RWMutex
		sensor        map[string]*sensorState
		sensorWorkers *device.SensorInfo
		registry      *sensorRegistry
		hassConfig    *hass.HassConfig
	}
	type args struct {
		ctx      context.Context
		updateCh chan interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "test adding worker",
			fields: fields{sensorWorkers: fakeWorkers},
			args:   args{ctx: ctx, updateCh: updateCh},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := &sensorTracker{
				mu:            tt.fields.mu,
				sensor:        tt.fields.sensor,
				sensorWorkers: tt.fields.sensorWorkers,
				registry:      tt.fields.registry,
				hassConfig:    tt.fields.hassConfig,
			}
			tracker.StartWorkers(tt.args.ctx, tt.args.updateCh)
		})
	}
}