package seed

// CAs creates Certificate Authority entries. The real CA types (DigiCert,
// SmallStep, NDES, Hydrant) need credentialed config that's meaningless on a
// dev Fleet; for now we record intent and skip. Once a "mock CA" type lands
// in Fleet for testing, this is the spot to wire it up.
func CAs(c Client, log Logger, count int) Result {
	res := Result{Entity: "cas"}
	for i := 0; i < count; i++ {
		log.Printf("ca #%d (planned) — mock CA creation not yet implemented", i+1)
		res.Skipped++
	}
	_ = c
	return res
}
