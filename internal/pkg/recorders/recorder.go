package recorders

type Recorder interface {
	Init() error
	Close() error
	Write(data []byte, isClientInput bool) error
}

