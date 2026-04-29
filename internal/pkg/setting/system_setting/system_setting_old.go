package system_setting

// Default empty; the frontend falls back to window.location.origin so the
// example endpoint URL on the landing page reflects whatever host the user
// connected to (matters once the deployment is exposed under a real domain
// instead of a fixed IP:port).
var ServerAddress = ""
var WorkerUrl = ""
var WorkerValidKey = ""
var WorkerAllowHttpImageRequestEnabled = false

func EnableWorker() bool {
	return WorkerUrl != ""
}
