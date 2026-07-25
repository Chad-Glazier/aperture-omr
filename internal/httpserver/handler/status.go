package handler

import "net/http"

func GetResourceUtilization(s ServerResources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get server resource utilization
		// - Memory (RSS bytes; total available memory)
		// - CPU (per core; cumulative)
		// - Disk
		//   - templates/scans stored (abstract data)
		//   - matrices/pictures stored + bytes (raw data)
		//   - bytes for database rows
		//   - total bytes and total available disk space

	}
}
