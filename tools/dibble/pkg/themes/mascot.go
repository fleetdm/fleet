package themes

// TapirSnout is the inline bullet character for progress lines.
const TapirSnout = ">·)~"

// tapirArt is the Braille-block tapir used in both the wizard banner and
// the README header. Small and Large are the same art today; kept as
// distinct exported names so callers can diverge later without churn.
const tapirArt = `
⠀⠀⠀⠀⠀⣀⣀⣤⣤⣤⣤⣤⠀⣀⣀⣀⠀⠀⠀⠀⠀⠀⡀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⣠⣴⣿⣿⣿⣿⣿⣿⣿⣿⡆⠸⣿⣿⣿⣷⣶⣤⣄⣾⣷⡄⠀⠀⠀⠀⠀⠀
⠀⢰⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠀⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣶⣤⡀⠀⠀⠀
⠀⣤⣤⣤⣈⡉⠛⢿⣿⣿⣿⣿⣿⡆⢸⣿⣿⣿⣿⣿⣿⣿⣿⣧⣽⣿⣷⣄⠀⠀
⠀⢿⠿⣿⣿⣿⣷⣤⡈⢻⣿⣿⣿⣇⠈⣿⣿⣿⣿⣿⣿⠿⣿⣿⣿⣿⣿⣿⡄⠀
⠀⠈⠀⢸⣿⣿⣿⣿⠇⠀⠛⠛⠛⠋⠀⢻⣿⣿⡟⢉⠀⠀⠈⠙⠛⠿⠏⣿⣷⠀
⠀⠀⢠⣿⣿⡿⠟⢁⡄⠀⠀⠀⠀⠀⠀⠈⣿⣿⡇⣾⡀⠀⠀⠀⠀⠀⠀⠸⠿⠀
⠀⠀⠸⣿⣿⠀⢸⣿⣇⠀⠀⠀⠀⠀⠀⠀⢹⣿⡇⠸⣧⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠙⠛⠃⠀⠛⠛⠀⠀⠀⠀⠀⠀⠀⠘⠛⠛⠀⠙⠃⠀⠀⠀⠀⠀⠀⠀⠀
`

// TapirSmall is the wizard banner art.
const TapirSmall = tapirArt

// TapirLarge is the README header art.
const TapirLarge = tapirArt
