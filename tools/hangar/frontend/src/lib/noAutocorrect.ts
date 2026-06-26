/// Spread into <input> / <textarea> to suppress macOS WebKit's
/// autocorrect, autocapitalize, and red-underline spell-check. Native
/// browser behavior we never want in this dev tool — typing a backup
/// name and getting "Backup Name" capitalized, or a commit-style label
/// flagged as a misspelling, is more friction than help.
///
/// Doesn't touch `autoComplete` — password / email fields still want
/// the password-manager hint set explicitly.
export const noAutocorrect = {
  spellCheck: false,
  autoCorrect: "off",
  autoCapitalize: "off",
} as const;
