// Copyright (c) 2023 Joshua Rich <joshua.rich@gmail.com>
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package sensors

import (
	"bytes"
	"encoding/json"

	"github.com/joshuar/go-hass-agent/internal/hass"
	"github.com/rs/zerolog/log"
)

// sensorState tracks the current state of a sensor, including the sensor value
// and whether it is registered/disabled in HA.
type sensorState struct {
	deviceClass hass.SensorDeviceClass
	stateClass  hass.SensorStateClass
	sensorType  hass.SensorType
	state       interface{}
	stateUnits  string
	attributes  interface{}
	icon        string
	name        string
	entityID    string
	category    string
	metadata    *sensorMetadata
}

type sensorMetadata struct {
	Registered bool `json:"Registered"`
	Disabled   bool `json:"Disabled"`
}

// sensorState implements hass.Sensor to represent a sensor in HA.

func (s *sensorState) DeviceClass() string {
	if s.deviceClass != 0 {
		return s.deviceClass.String()
	} else {
		return ""
	}
}

func (s *sensorState) StateClass() string {
	if s.stateClass != 0 {
		return s.stateClass.String()
	} else {
		return ""
	}
}

func (s *sensorState) Type() string {
	if s.sensorType != 0 {
		return s.sensorType.String()
	} else {
		return hass.TypeSensor.String()
	}
}

func (s *sensorState) Icon() string {
	return s.icon
}

func (s *sensorState) Name() string {
	return s.name
}

func (s *sensorState) State() interface{} {
	if s.state != nil {
		return s.state
	} else {
		return "Unknown"
	}
}

func (s *sensorState) Attributes() interface{} {
	return s.attributes
}

func (s *sensorState) UniqueID() string {
	return s.entityID
}

func (s *sensorState) UnitOfMeasurement() string {
	return s.stateUnits
}

func (s *sensorState) EntityCategory() string {
	return s.category
}

func (s *sensorState) Disabled() bool {
	if s.metadata != nil {
		return s.metadata.Disabled
	} else {
		return false
	}

}

func (s *sensorState) Registered() bool {
	if s.metadata != nil {
		return s.metadata.Registered
	} else {
		return false
	}
}

// sensorState implements hass.Request so its data can be sent to the HA API

func (sensor *sensorState) RequestType() hass.RequestType {
	if sensor.metadata.Registered {
		return hass.RequestTypeUpdateSensorStates
	}
	return hass.RequestTypeRegisterSensor
}

func (sensor *sensorState) RequestData() interface{} {
	return hass.MarshalSensorData(sensor)
}

func (sensor *sensorState) ResponseHandler(rawResponse bytes.Buffer) {
	switch {
	case rawResponse.Len() == 0 || rawResponse.String() == "{}":
		log.Debug().Caller().
			Msgf("No response for %s request. Likely problem with request data.", sensor.name)
	default:
		var r interface{}
		err := json.Unmarshal(rawResponse.Bytes(), &r)
		if err != nil {
			log.Debug().Caller().Err(err).
				Msg("Could not unmarshal response.")
			return
		}
		response := r.(map[string]interface{})
		if v, ok := response["success"]; ok {
			if v.(bool) && !sensor.metadata.Registered {
				sensor.metadata.Registered = true
				log.Debug().Caller().
					Msgf("Sensor %s registered in HA.",
						sensor.Name())
			}
		}
		if v, ok := response[sensor.entityID]; ok {
			status := v.(map[string]interface{})
			if !status["success"].(bool) {
				error := status["error"].(map[string]interface{})
				log.Debug().Caller().
					Msgf("Could not update sensor %s, %s: %s",
						sensor.Name(),
						error["code"],
						error["message"])
			} else {
				log.Debug().Caller().
					Msgf("Sensor %s updated (%s). State is now: %v %s",
						sensor.Name(),
						sensor.UniqueID(),
						sensor.State(),
						sensor.UnitOfMeasurement())
			}
			if _, ok := status["is_disabled"]; ok {
				sensor.metadata.Disabled = true
			} else if sensor.metadata.Disabled {
				sensor.metadata.Disabled = false
			}
		}
	}
}

func marshalSensorState(s hass.SensorUpdate) *sensorState {
	return &sensorState{
		entityID:    s.ID(),
		name:        s.Name(),
		deviceClass: s.DeviceClass(),
		stateClass:  s.StateClass(),
		sensorType:  s.SensorType(),
		state:       s.State(),
		attributes:  s.Attributes(),
		icon:        s.Icon(),
		stateUnits:  s.Units(),
		category:    s.Category(),
	}
}
