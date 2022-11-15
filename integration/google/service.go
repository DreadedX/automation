package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"google.golang.org/api/homegraph/v1"
)

type ExecuteResponse struct {
	UpdatedState   DeviceState
	UpdatedDevices []string
	OfflineDevices []string
	// The key is the errorCode that is associated with the devices
	FailedDevices map[string]struct {
		Devices []string
	}
}

type Provider interface {
	Sync(context.Context, string) ([]*Device, error)
	Query(context.Context, string, []DeviceHandle) (map[string]DeviceState, error)
	Execute(context.Context, string, []Command) (*ExecuteResponse, error)
}

type Service struct {
	provider      Provider
	deviceService *homegraph.DevicesService
}

func NewService(provider Provider, service *homegraph.Service) *Service {
	return &Service{
		provider:      provider,
		deviceService: homegraph.NewDevicesService(service),
	}
}

func (s *Service) RequestSync(ctx context.Context, userID string) error {
	call := s.deviceService.RequestSync(&homegraph.RequestSyncDevicesRequest{
		AgentUserId: userID,
	})

	call.Context(ctx)
	resp, err := call.Do()
	if err != nil {
		return err
	}

	if resp.ServerResponse.HTTPStatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("sync failed: %d", resp.ServerResponse.HTTPStatusCode))
	}

	return nil
}

func (s *Service) ReportState(ctx context.Context, userID string, states map[string]DeviceState) error {
	j, err := json.Marshal(states)
	if err != nil {
		return err
	}

	call := s.deviceService.ReportStateAndNotification(&homegraph.ReportStateAndNotificationRequest{
		AgentUserId: userID,
		EventId:     uuid.New().String(),
		RequestId:   uuid.New().String(),
		Payload: &homegraph.StateAndNotificationPayload{
			Devices: &homegraph.ReportStateAndNotificationDevice{
				States:        j,
			},
		},
	})

	call.Context(ctx)
	resp, err := call.Do()
	if err != nil {
		return err
	}

	if resp.ServerResponse.HTTPStatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("report failed: %d", resp.ServerResponse.HTTPStatusCode))
	}

	return nil
}
