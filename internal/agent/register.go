// Copyright (c) 2023 Joshua Rich <joshua.rich@gmail.com>
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/joshuar/go-hass-agent/internal/device"
	"github.com/joshuar/go-hass-agent/internal/hass"
	"github.com/joshuar/go-hass-agent/internal/linux"
	"github.com/rs/zerolog/log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	validate "github.com/go-playground/validator/v10"
)

func NewRegistration() *hass.RegistrationHost {
	return &hass.RegistrationHost{
		Server: binding.NewString(),
		Token:  binding.NewString(),
		UseTLS: binding.NewBool(),
	}
}

func findServers(ctx context.Context) binding.StringList {

	serverList := binding.NewStringList()

	// add http://localhost:8123 to the list of servers as a fall-back/default
	// option
	serverList.Append("localhost:8123")

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Warn().Msgf("Failed to initialize resolver:", err.Error())
	} else {
		entries := make(chan *zeroconf.ServiceEntry)
		go func(results <-chan *zeroconf.ServiceEntry) {
			for entry := range results {
				server := entry.AddrIPv4[0].String() + ":" + fmt.Sprint(entry.Port)
				serverList.Append(server)
				log.Debug().Caller().
					Msg("Found a HA instance via mDNS")
			}
		}(entries)

		log.Info().Msg("Looking for Home Assistant instances on the network...")
		searchCtx, searchCancel := context.WithTimeout(ctx, time.Second*5)
		defer searchCancel()
		err = resolver.Browse(searchCtx, "_home-assistant._tcp", "local.", entries)
		if err != nil {
			log.Warn().Msgf("Failed to browse:", err.Error())
		}

		<-searchCtx.Done()
	}
	return serverList
}

func (agent *Agent) requestRegistrationInfoUI(ctx context.Context) *hass.RegistrationHost {

	registrationInfo := NewRegistration()

	var wg sync.WaitGroup

	s := findServers(ctx)
	allServers, _ := s.Get()

	w := agent.app.NewWindow(translator.Translate("App Registration"))

	tokenSelect := widget.NewEntryWithData(registrationInfo.Token)

	autoServerSelect := widget.NewSelect(allServers, func(s string) {
		registrationInfo.Server.Set(s)
	})

	manualServerEntry := widget.NewEntryWithData(registrationInfo.Server)
	manualServerEntry.Validator = newHostPort()
	manualServerEntry.Disable()
	manualServerSelect := widget.NewCheck("", func(b bool) {
		switch b {
		case true:
			manualServerEntry.Enable()
			autoServerSelect.Disable()
		case false:
			manualServerEntry.Disable()
			autoServerSelect.Enable()
		}
	})

	tlsSelect := widget.NewCheckWithData("", registrationInfo.UseTLS)

	form := widget.NewForm(
		widget.NewFormItem(translator.Translate("Token"), tokenSelect),
		widget.NewFormItem(translator.Translate("Auto-discovered Servers"), autoServerSelect),
		widget.NewFormItem(translator.Translate("Use Custom Server?"), manualServerSelect),
		widget.NewFormItem(translator.Translate("Manual Server Entry"), manualServerEntry),
		widget.NewFormItem(translator.Translate("Use TLS?"), tlsSelect),
	)
	form.OnSubmit = func() {
		s, _ := registrationInfo.Server.Get()
		log.Debug().Caller().
			Msgf("User selected server %s", s)

		w.Close()
		wg.Done()
	}
	form.OnCancel = func() {
		registrationInfo = nil
		wg.Done()
	}

	w.SetContent(container.New(layout.NewVBoxLayout(),
		widget.NewLabel(
			translator.Translate(
				"As an initial step, this app will need to log into your Home Assistant server and register itself.\nPlease enter the relevant details for your Home Assistant server url/port and a long-lived access token.")),
		form,
	))
	w.Show()
	wg.Add(1)
	wg.Wait()
	w.Close()
	return registrationInfo
}

func (agent *Agent) saveRegistration(r *hass.RegistrationResponse, h *hass.RegistrationHost) {
	host, _ := h.Server.Get()
	useTLS, _ := h.UseTLS.Get()
	agent.SetPref("Host", host)
	agent.SetPref("UseTLS", useTLS)
	token, _ := h.Token.Get()
	agent.SetPref("Token", token)
	agent.SetPref("Version", agent.Version)
	if r.CloudhookURL != "" {
		agent.SetPref("CloudhookURL", r.CloudhookURL)
	}
	if r.RemoteUIURL != "" {
		agent.SetPref("RemoteUIURL", r.RemoteUIURL)
	}
	if r.Secret != "" {
		agent.SetPref("Secret", r.Secret)
	}
	if r.WebhookID != "" {
		agent.SetPref("WebhookID", r.WebhookID)
	}
	// ! https://github.com/fyne-io/fyne/issues/3170
	time.Sleep(110 * time.Millisecond)
}

func (agent *Agent) runRegistrationWorker(ctx context.Context, getRegistrationInfo func(context.Context) *hass.RegistrationHost) error {
	thisDevice := linux.NewDevice(ctx, Name, Version)
	agent.SetPref("DeviceID", thisDevice.DeviceID())
	agent.SetPref("DeviceName", thisDevice.DeviceName())
	registrationHostInfo := getRegistrationInfo(ctx)
	if registrationHostInfo != nil {
		registrationRequest := device.GenerateRegistrationRequest(thisDevice)
		appRegistrationInfo := hass.RegisterWithHass(registrationHostInfo, registrationRequest)
		if appRegistrationInfo != nil {
			agent.saveRegistration(appRegistrationInfo, registrationHostInfo)
			return nil
		} else {
			return errors.New("registration failed")
		}
	} else {
		return errors.New("problem getting registration information")
	}
}

// newHostPort is a custom fyne validator that will validate a string is a
// valid hostname:port combination
func newHostPort() fyne.StringValidator {
	v := validate.New()
	return func(text string) error {
		if err := v.Var(text, "hostname_port"); err != nil {
			return errors.New("you need to specify a valid hostname:port combination")
		}
		return nil
	}
}
