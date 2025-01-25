package consts

const (
	DocPathPrefix    = "documents/"
	AvatarPathPrefix = "avatars/"

	UserAlreadyExist     = 5000
	UserNotExist         = 5001
	ShouldBindFailed     = 5002
	RegisterFailed       = 5003
	MysqlQueryFailed     = 5004
	RedisQueryFailed     = 5005
	PasswordHashedWrong  = 5006
	PasswordCompareWrong = 5007
	AvatarQueryFailed    = 5008
	DocxQueryFailed      = 5009
	NotFoundUserInMiddle = 5010
	OpenFileWrong        = 5011
	CloseFileWrong       = 5012
	UploadFileWrong      = 5013
	DeleteFileWrong      = 5014
	GetFileWrong         = 5014
	MysqlSaveWrong       = 5015
	FilenameMissing      = 5016
	FileNotFind          = 5017
	GenerateTokenFailed  = 5018
	VisibilityNotCorrect = 5019
	VisibilityWrong      = 5020
	UserCannotChangeFile = 5021
	FileResponseWrong    = 5022
	CreateFileAccessFail = 5023
)
