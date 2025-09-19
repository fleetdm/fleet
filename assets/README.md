# assets Directory

This directory is used by the frontend build tool to store generated frontend build assets, such as JavaScript bundles, CSS files, and other static resources (images, fonts, etc.) that are produced during the build process.

## Key Points

- **Do not commit generated files:** All files in this directory are generated automatically and are git-ignored. They should not be committed to version control.
- **Not for static files:** Do not place unrelated static files (like PDFs, config profiles, or documentation) here. Use a separate directory for non-UI assets.
- **Regenerated on build:** These files are recreated every time you run the frontend build process.
