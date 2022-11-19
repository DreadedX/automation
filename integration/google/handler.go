package google

import (
	"automation/device"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/jellydator/ttlcache/v3"
)

type DeviceInterface interface {
	device.Basic

	Sync() *Device
	Query() DeviceState
	Execute(execution Execution, updatedState *DeviceState) (errCode string, online bool)
}

// https://developers.google.com/assistant/smarthome/reference/intent/sync
type syncResponse struct {
	RequestID string `json:"requestId"`
	Payload   struct {
		UserID      string    `json:"agentUserId"`
		ErrorCode   string    `json:"errorCode,omitempty"`
		DebugString string    `json:"debugString,omitempty"`
		Devices     []*Device `json:"devices"`
	} `json:"payload"`
}

// https://developers.google.com/assistant/smarthome/reference/intent/query
type queryResponse struct {
	RequestID string `json:"requestId"`
	Payload   struct {
		ErrorCode   string                   `json:"errorCode,omitempty"`
		DebugString string                   `json:"debugString,omitempty"`
		Devices     map[string]DeviceState `json:"devices"`
	} `json:"payload"`
}

type executeRespPayload struct {
	IDs       []string    `json:"ids"`
	Status    Status      `json:"status"`
	ErrorCode string      `json:"errorCode,omitempty"`
	States    DeviceState `json:"states,omitempty"`
}

type executeResponse struct {
	RequestID string `json:"requestId"`
	Payload   struct {
		ErrorCode   string               `json:"errorCode,omitempty"`
		DebugString string               `json:"debugString,omitempty"`
		Commands    []executeRespPayload `json:"commands,omitempty"`
	} `json:"payload"`
}

// @TODO We can also implement this as a cache loader function
// Note sure how to report the correct errors in that case?
func (s *Service) getUser(authorization string) (string, int) {
	// @TODO Make oids url configurable

	cached := s.cache.Get(authorization)
	if cached != nil {
		return cached.Value(), http.StatusOK
	}

	req, err := http.NewRequest("GET", "https://login.huizinga.dev/api/oidc/userinfo", nil)
	if err != nil {
		log.Println("Failed to make request to to login server")
		return "", http.StatusInternalServerError
	}

	req.Header.Set("Authorization", authorization)
	client := &http.Client{}
	resp, err := client.Do(req)

	// If we get something other than 200, error out
	if resp.StatusCode != http.StatusOK {
		log.Println("Not logged in...")
		return "", resp.StatusCode
	}

	// Get the preferred_username from the userinfo
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to read body")
		return "", resp.StatusCode
	}

	var body struct {
		PreferredUsername string `json:"preferred_username"`
	}

	err = json.Unmarshal(bodyBytes, &body)
	if err != nil {
		log.Println("Failed to marshal body")
		return "", http.StatusInternalServerError
	}

	if len(body.PreferredUsername) == 0 {
		log.Println("Received empty username from userinfo endpoint")
		return "", http.StatusInternalServerError
	}

	s.cache.Set(authorization, body.PreferredUsername, ttlcache.DefaultTTL)

	return body.PreferredUsername, http.StatusOK
}

func (s *Service) FullfillmentHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// Check if we are logged in
	var userID string
	if auth, ok := r.Header["Authorization"]; ok && len(auth) > 0 {
		var statusCode int
		userID, statusCode = s.getUser(auth[0])
		if statusCode != http.StatusOK {
			w.WriteHeader(statusCode)
			return
		}
	} else {
		log.Println("No authorization provided")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	fullfimentReq := &FullfillmentRequest{}
	err := json.NewDecoder(r.Body).Decode(&fullfimentReq)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("JSON Deserialization failed"))
		return
	}

	if len(fullfimentReq.Inputs) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Unsupported number of inputs"))
		return
	}

	switch fullfimentReq.Inputs[0].Intent {
	case IntentSync:
		devices, err := s.provider.Sync(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Failed to sync"))
		}

		syncResp := &syncResponse{
			RequestID: fullfimentReq.RequestID,
		}
		syncResp.Payload.UserID = userID
		syncResp.Payload.Devices = devices

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(syncResp)
		if err != nil {
			log.Println("Error serializing", err)
		}

	case IntentQuery:
		states, err := s.provider.Query(r.Context(), userID, fullfimentReq.Inputs[0].Query.Devices)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Failed to sync"))
		}

		queryResp := &queryResponse{
			RequestID: fullfimentReq.RequestID,
		}
		queryResp.Payload.Devices = states

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(queryResp)
		if err != nil {
			log.Println("Error serializing", err)
		}

	case IntentExecute:
		response, err := s.provider.Execute(r.Context(), userID, fullfimentReq.Inputs[0].Execute.Commands)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Failed to sync"))
		}

		executeResp := &executeResponse{
			RequestID: fullfimentReq.RequestID,
		}

		if len(response.UpdatedDevices) > 0 {
			c := executeRespPayload{
				Status: StatusSuccess,
				States: response.UpdatedState,
			}

			for _, id := range response.UpdatedDevices {
				c.IDs = append(c.IDs, id)
			}

			executeResp.Payload.Commands = append(executeResp.Payload.Commands, c)
		}

		if len(response.OfflineDevices) > 0 {
			c := executeRespPayload{
				Status: StatusOffline,
			}

			for _, id := range response.UpdatedDevices {
				c.IDs = append(c.IDs, id)
			}

			executeResp.Payload.Commands = append(executeResp.Payload.Commands, c)
		}

		for errCode, details := range response.FailedDevices {
			c := executeRespPayload{
				Status: StatusError,
				ErrorCode: errCode,
			}

			for _, id := range details.Devices {
				c.IDs = append(c.IDs, id)
			}

			executeResp.Payload.Commands = append(executeResp.Payload.Commands, c)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(executeResp)
		if err != nil {
			log.Println("Error serializing", err)
		}

	default:
		log.Println("Intent is not implemented")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Not implemented for now"))
	}
}
