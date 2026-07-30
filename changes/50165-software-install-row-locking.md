- Fixed 502 errors and timeouts on the software install endpoints caused by the hourly Fleet-maintained apps sync taking exclusive row locks on the entire `software` and `software_titles` tables while normalizing software names.
- Fixed a macOS Fleet-maintained app's name being applied to the iOS and iPadOS apps that share its bundle identifier.
- Fixed macOS Fleet-maintained app software names not being corrected when the catalog refresh failed, by moving the correction to its own schedule instead of running it as a step of that refresh.

