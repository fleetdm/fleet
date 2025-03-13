# custom-package-parser

Tool to extract the metadata of software packages (same way Fleet would extract metadata on uploads).
This tool was used to determine accuracy of Fleet's processing of software packages (with the most used/popular apps) (see [tests.md](./tests.md)).

Using a local file:
```sh
go run ./tools/custom-package-parser -path ~/Downloads/MicrosoftTeams.pkg
- Name: 'Microsoft Teams.app'
- Bundle Identifier: 'com.microsoft.teams2'
- Package IDs: 'com.microsoft.teams2,com.microsoft.package.Microsoft_AutoUpdate.app,com.microsoft.MSTeamsAudioDevice'
```

Using a URL:
```sh
go run ./tools/custom-package-parser -url https://downloads.1password.com/win/1PasswordSetup-latest.msi
- Name: '1Password'
- Bundle Identifier: ''
- Package IDs: '{321BD799-2490-40D7-8A88-6888809FA681}'
```