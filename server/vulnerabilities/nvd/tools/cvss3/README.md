# CVSS3

This package implements a [CVSS v3 specification](https://www.first.org/cvss/specification-document) and provides functions for serialization and deserialization of vectors as well as score calculation (base, temporal and environmental).

## Usage

```golang
vec, err := cvss3.VectorFromString("CVSS:3.0/AV:L/AC:H/PR:H/UI:R/S:C/C:L/I:H/A:L/E:P/RL:W/RC:R/CR:M/IR:H/AR:L/MAV:N/MAC:H/MPR:L/MUI:R/MS:C/MC:L/MA:N")
if err != nil {
    panic(err)
}
if err := vec.Validate(); err != nil {
    panic(err)
}

fmt.Println(vec, vec.BaseScore(), vec.TemporalScore(), vec.EnvironmentalScore())
// CVSS:3.0/AV:L/AC:H/PR:H/UI:R/S:C/C:L/I:H/A:L/E:P/RL:W/RC:R/CR:M/IR:H/AR:L/MAV:N/MAC:H/MPR:L/MUI:R/MS:C/MC:L/MA:N 6.4 5.7 7.1

vec.EnvironmentalMetrics.ModifiedScope = ScopeUnchanged
fmt.Println(vec, vec.BaseScore(), vec.TemporalScore(), vec.EnvironmentalScore())
// CVSS:3.0/AV:L/AC:H/PR:H/UI:R/S:C/C:L/I:H/A:L/E:P/RL:W/RC:R/CR:M/IR:H/AR:L/MAV:N/MAC:H/MPR:L/MUI:R/MS:U/MC:L/MA:N 6.4, 5.7, 6.1
```
