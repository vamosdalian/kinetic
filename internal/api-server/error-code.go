package apiserver

const (
	ErrorCodeInvalidRequest  = "INVALID_REQUEST"
	ErrorCodeInternalError   = "INTERNAL_ERROR"
	ErrorCodeInvalidWorkflow = "INVALID_WORKFLOW"
	ErrorCodeUnauthorized    = "UNAUTHORIZED"
	ErrorCodeForbidden       = "FORBIDDEN"

	ErrorCodeWorkflowNotFound             = "WORKFLOW_NOT_FOUND"
	ErrorCodeUserNotFound                 = "USER_NOT_FOUND"
	ErrorCodeInvalidCredentials           = "INVALID_CREDENTIALS"
	ErrorCodeUserAlreadyExists            = "USER_ALREADY_EXISTS"
	ErrorCodeBootstrapAdminDeleteForbidden = "BOOTSTRAP_ADMIN_DELETE_FORBIDDEN"
)
