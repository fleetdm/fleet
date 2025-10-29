# Creating Icons for Fleet-maintained Apps

App icons for the Fleet server and [fleetdm.com software catalog](https://fleetdm.com/software-catalog) can be generated using the following script on macOS using an associated `.app` bundle.

## Usage

```bash
bash tools/software/icons/generate-icons.sh -a /path/to/App.app -s slug-name
```

- `-a`: Path to the `.app` bundle (e.g. `/Applications/Safari.app`)
- `-s`: Slug name for the Fleet-maintained app.  The portion before the slash will be used in the output filenames.

## Example

```bash
bash tools/software/icons/generate-icons.sh -a /Applications/Google\ Chrome.app -s "google-chrome/darwin"
```

This will generate two files:

- `frontend/pages/SoftwarePage/components/icons/GoogleChrome.tsx` – the SVG React component
- `website/assets/images/app-icon-google-chrome-60x60@2x.png` – the 128x128 PNG used on the website

## Notes

- The SVG generated is embedded with a base64-encoded 32×32 version of the app's 128×128 PNG icon.
- The TSX component name is derived from the app’s name (e.g. `Google Chrome.app` → `GoogleChrome.tsx`).
- The script ensures consistent formatting and naming conventions across icon components.
