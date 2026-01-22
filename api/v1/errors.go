package v1

var (
	// common errors
	ErrSuccess             = newError(0, "ok")
	ErrBadRequest          = newError(400, "bad request")
	ErrUnauthorized        = newError(401, "unauthorized")
	ErrNotFound            = newError(404, "not found")
	ErrInternalServerError = newError(500, "internal server error")

	// more biz errors
	ErrEmailAlreadyUse   = newError(1001, "The email is already in use.")
	ErrUsernameAlreadyUse = newError(1002, "The username is already in use.")

	// vm create mode errors
	ErrInvalidCreateMode = newError(400, "invalid create mode")
	
	// template management errors
	ErrStorageNotFound        = newError(2001, "storage not found")
	ErrNodeNotFound           = newError(2002, "node not found")
	ErrFileUploadFailed       = newError(2003, "file upload failed")
	ErrTemplateImportFailed   = newError(2004, "template import failed")
	ErrSharedStorageNoSync    = newError(2005, "shared storage does not need sync")
	ErrInvalidOperation       = newError(2006, "invalid operation")
)
