package google

import (
	"encoding/json"
	"log"
	"net/http"
)

type DeviceInterface interface {
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

func (s *Service) FullfillmentHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// Check if we are logged in
	{
		req, err := http.NewRequest("GET", "https://login.huizinga.dev/api/oidc/userinfo", nil)
		if err != nil {
			log.Println("Failed to make request to to login server")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if len(r.Header["Authorization"]) > 0 {
			req.Header.Set("Authorization", r.Header["Authorization"][0])
		}
		client := &http.Client{}
		resp, err := client.Do(req)

		// If we get something other than 200, error out
		if resp.StatusCode != http.StatusOK {
			log.Println("Not logged in...")
			w.WriteHeader(resp.StatusCode)
			return
		}
	}

	// @TODO Make sure we receive content type json
	// @TODO Get this from userinfo, currently the scope is not set up properly to actually receive the username
	userID := "Dreaded_X"

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
