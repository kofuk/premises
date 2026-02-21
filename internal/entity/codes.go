package entity

const (
	ErrBadRequest       ErrorCode = 1
	ErrInternal         ErrorCode = 2
	ErrCredential       ErrorCode = 3
	ErrServerRunning    ErrorCode = 4
	ErrServerNotRunning ErrorCode = 5
	ErrRemote           ErrorCode = 6
	ErrInvalidConfig    ErrorCode = 7
	ErrPasswordRule     ErrorCode = 8
	ErrDupUserName      ErrorCode = 9
	ErrRequiresAuth     ErrorCode = 10
	ErrBackup           ErrorCode = 11
	ErrAgain            ErrorCode = 12
)

const (
	InfoSnapshotDone     InfoCode = 1
	InfoSnapshotError    InfoCode = 2
	InfoNoSnapshot       InfoCode = 3
	InfoErrRunnerPrepare InfoCode = 100
	InfoErrRunnerStop    InfoCode = 101
)

const (
	EventShutdown      EventCode = 1
	EventSysInit       EventCode = 2
	EventGameDownload  EventCode = 3
	EventWorldDownload EventCode = 4
	EventWorldPrepare  EventCode = 5
	EventWorldUpload   EventCode = 6
	EventLoading       EventCode = 7
	EventRunning       EventCode = 8
	EventStopping      EventCode = 9
	EventCrashed       EventCode = 10
	EventClean         EventCode = 11
	EventStopped       EventCode = 100
	EventCreateRunner  EventCode = 101
	EventWaitConn      EventCode = 102
	EventConnLost      EventCode = 103
	EventStopRunner    EventCode = 104
	EventManualSetup   EventCode = 105
)

// Event codes that should be provided UI to retry.
const (
	EventGameErr   EventCode = 50
	EventWorldErr  EventCode = 51
	EventLaunchErr EventCode = 52
)
