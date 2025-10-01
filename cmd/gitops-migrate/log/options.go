package log

var Options options

type options uint8

const (
	OptWithLevel options = 1 << iota
	OptWithCaller
)

func (o options) WithLevel() bool  { return o&OptWithLevel == OptWithLevel }
func (o *options) SetWithLevel()   { *o |= OptWithLevel }
func (o *options) UnsetWithLevel() { *o &^= OptWithLevel }

func (o options) WithCaller() bool  { return o&OptWithCaller == OptWithCaller }
func (o *options) SetWithCaller()   { *o |= OptWithCaller }
func (o *options) UnsetWithCaller() { *o &^= OptWithCaller }
