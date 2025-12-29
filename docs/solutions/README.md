# Solutions

## Best Practices

### General

- Name the file what the profile does.
  - For example, instead of `googlePlayProtectVerifyApps.json` (the name of the Android policy for this control), describe what it does: `enforce-google-play-protect.json`.
- Use kebab case in file names, with all letters in lowercase.
  - Instead of `passwordPolicy.json`, use `password-policy.json`.
- Be sure to end files with an empty newline.


### symlinks

If a solution is applicable to multiple platforms, keep the original in the main platform directory and symlink it to the other platforms. For example, if an Apple configuration profile can be used on both macOS and iOS, use macOS as the source, and create a symlink in the iOS directory.

– `cd docs/solutions/ios-ipados/configuration-profiles/`
  - Note that this is the destination that we want the symlink to be in.
– `ln -s ../../macos/configuration-profiles/my-profile.mobileconfig .`
  - The `.` here at the end means the current directory, and will use the same file name as the original (which is what we want).
– `git add profile.mobileconfig`
- `git commit`
