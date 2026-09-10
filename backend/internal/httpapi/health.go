package httpapi

import "net/http"

type healthResponse struct {
	Status     string `json:"status"`
	ServerTime int64  `json:"server_time"`
}

// handleHealth tidak memerlukan autentikasi dan menyertakan jam server,
// sehingga perangkat dapat mendeteksi jamnya sendiri meleset sebelum
// mengirim apa pun.
func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:     "ok",
		ServerTime: a.now().Unix(),
	})
}
