# Creating Icons for Fleet-maintained Apps

App icons for the Fleet server and [fleetdm.com software catalog](https://fleetdm.com/software-catalog) can be generated using the following script on macOS using an associated `.app` bundle.

## Usage

```bash
bash tools/software/icons/generate-icons.sh -s slug-name [-a /path/to/App.app | -i /path/to/icon.png]
```

- `-s`: Slug name for the Fleet-maintained app (required). The portion before the slash will be used in the output filenames.
- `-a`: Path to the `.app` bundle (e.g. `/Applications/Safari.app`). Required if `-i` is not provided.
- `-i`: Path to a PNG icon file. Required if `-a` is not provided. The icon will be resized to 128x128 if larger.

## Examples

Using an app bundle:
```bash
bash tools/software/icons/generate-icons.sh -a /Applications/Google\ Chrome.app -s "google-chrome/darwin"
```

Using a PNG file directly:
```bash
bash tools/software/icons/generate-icons.sh -i /path/to/icon.png -s "company-portal/windows"
```

This will generate two files:

- `frontend/pages/SoftwarePage/components/icons/GoogleChrome.tsx` – the SVG React component
- `website/assets/images/app-icon-google-chrome-60x60@2x.png` – the 128x128 PNG used on the website

## Notes

- The SVG generated is embedded with a base64-encoded 32×32 version of the app's 128×128 PNG icon.
- The TSX component name is derived from the app's name (e.g. `Google Chrome.app` → `GoogleChrome.tsx`).
- The script ensures consistent formatting and naming conventions across icon components.
- **The script automatically adds the import statement and map entry to `frontend/pages/SoftwarePage/components/icons/index.ts`**, so you don't need to manually update the index file. The app name used in the map is extracted from the app's `Info.plist` (`CFBundleName` or `CFBundleDisplayName`).
