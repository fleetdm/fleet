# CVSS2

This package implements a [CVSS v2 specification](https://www.first.org/cvss/v2/guide) and provides functions for serialization and deserialization of vectors as well as score calculation (base, temporal and environmental).

## Usage

```golang
vec, err := cvss2.VectorFromString("(AV:N/AC:M/Au:M/C:P/I:N/A:N/E:F/RL:W/RC:UR/CDP:LM/TD:M/CR:M/IR:H/AR:M)")
if err != nil {
    panic(err)
}
if err := vec.Validate(); err != nil {
    panic(err)
}

fmt.Println(vec, vec.BaseScore(), vec.TemporalScore(), vec.EnvironmentalScore())
// (AV:N/AC:M/Au:M/C:P/I:N/A:N/E:F/RL:W/RC:UR/CDP:LM/TD:M/CR:M/IR:H/AR:M) 2.8 2.4 3.5

vec.BaseMetrics.Authentification = AuthentificationSingle
fmt.Println(vec, vec.BaseScore(), vec.TemporalScore(), vec.EnvironmentalScore())
// (AV:N/AC:M/Au:S/C:P/I:N/A:N/E:F/RL:W/RC:UR/CDP:LM/TD:M/CR:M/IR:H/AR:M) 3.5 2.9 6.8
```
