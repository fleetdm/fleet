package api

// RegisterKindService registers a kind of end user notification. A kind is
// added this way and nowhere else: core resolves the kind column through the
// registry rather than switching on it, so adding one touches no delivery
// code.
type RegisterKindService interface {
	RegisterKind(kind NotificationKind)
}
