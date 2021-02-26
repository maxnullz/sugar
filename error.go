package sugar

type Errorz struct {
	ID      uint16
	Message string
}

func (r *Errorz) Error() string {
	return r.Message
}

var IDErrMap = map[uint16]error{}
var ErrIDMap = map[error]uint16{}

func NewError(str string, id uint16) *Errorz {
	err := &Errorz{id, str}
	IDErrMap[id] = err
	ErrIDMap[err] = id
	return err
}

var (
	ErrOk             = NewError("success", 0)
	ErrPBPack         = NewError("pb pack error", 1)
	ErrPBUnPack       = NewError("pb unpack error", 2)
	ErrJSONPack       = NewError("json pack error", 3)
	ErrJSONUnPack     = NewError("json unpack error", 4)
	ErrCmdUnPack      = NewError("cmd parse error", 5)
	ErrMsgLenTooLong  = NewError("message too long", 6)
	ErrMsgLenTooShort = NewError("message too short", 7)
	ErrDBDataType     = NewError("bad db type", 8)

	ErrErrIDNotFound = NewError("unknown error code", 255)
)

var MinUserError = 256

func GetError(id uint16) error {
	if e, ok := IDErrMap[id]; ok {
		return e
	}
	return ErrErrIDNotFound
}

func GetErrID(err error) uint16 {
	if id, ok := ErrIDMap[err]; ok {
		return id
	}
	return ErrIDMap[ErrErrIDNotFound]
}
