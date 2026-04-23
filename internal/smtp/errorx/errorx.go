package smtp_errorx

import "errors"

var (
	ErrConsumerNotFound  = errors.New("smtp: consumer not found")
	ErrTemplateNotFound  = errors.New("smtp: template not found")
	ErrGatewayNotFound   = errors.New("smtp: gateway not found")
	ErrEndpointNotFound  = errors.New("smtp: endpoint not found")
	ErrInvalidUserID     = errors.New("smtp: invalid user id")
	ErrInvalidConsumerID = errors.New("smtp: invalid consumer id")
	ErrInvalidTemplateID = errors.New("smtp: invalid template id")
	ErrInvalidGatewayID  = errors.New("smtp: invalid gateway id")
	ErrInvalidEndpointID = errors.New("smtp: invalid endpoint id")

	ErrUnauthorized      = errors.New("smtp: unauthorized")
	ErrWorkspaceRequired = errors.New("smtp: workspace is required")
	ErrZoneRequired      = errors.New("smtp: zone is required")
	ErrInvalidResource   = errors.New("smtp: invalid resource")
	ErrWorkspaceMismatch = errors.New("smtp: workspace mismatch")
	ErrZoneMismatch      = errors.New("smtp: zone mismatch")
	ErrTemplateConflict  = errors.New("smtp: template conflict")
	ErrRuntimeInvalid    = errors.New("smtp: runtime invalid")
	ErrDataPlaneNotFound = errors.New("smtp: dataplane not found")
	ErrDataPlaneNotReady = errors.New("smtp: dataplane not ready")
)
