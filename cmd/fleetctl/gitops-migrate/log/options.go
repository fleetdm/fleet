package log

var Options options

type options uint8

const (
	OptWithLevel options = 1 << iota
	OptWithCaller
)

func (self options) WithLevel() bool  { return self&OptWithLevel == OptWithLevel }
func (self *options) SetWithLevel()   { *self |= OptWithLevel }
func (self *options) UnsetWithLevel() { *self &^= OptWithLevel }

func (self options) WithCaller() bool  { return self&OptWithCaller == OptWithCaller }
func (self *options) SetWithCaller()   { *self |= OptWithCaller }
func (self *options) UnsetWithCaller() { *self &^= OptWithCaller }
