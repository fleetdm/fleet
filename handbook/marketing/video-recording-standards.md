# Video recording standards

Standards for recording sprint demos, product demos, webinars, and other video content for Fleet.

## How Fleet videos are delivered

Fleet publishes finished videos at 1920x1080 (1080p). That's the canvas the viewer sees, regardless of what you recorded on.

Two settings determine whether your recording is usable:

- Window size — how large the application window is on your screen while you record. This determines how big UI elements appear in the final video.
- Capture resolution — the pixel dimensions of the recording file. This determines how much detail editors have to work with.

These are independent. A window sized for an ultrawide display will produce unreadable UI in the final 1080p video no matter how high the capture resolution is. Size the window first, then choose your capture resolution.

The settings below live in different places: window size is set in the application itself or in your OS window manager; capture resolution is set in your screen recording tool.

<details style="background: #f6f8fa; border: 1px solid #d0d7de; border-radius: 6px; padding: 8px 16px;">
<summary style="cursor: pointer; font-weight: 600;">Some illustration of the difference</summary>

<table>
<tr><td>Terminal. Default font + window size. 3440x1440 capture (full desktop).<br><img src="https://raw.githubusercontent.com/fleetdm/fleet/main/handbook/marketing/images/terminal-default-font-size-full-desktop-capture.png" width="800" alt="terminal-default-font-size-full-desktop-capture.png"></td></tr>
<tr><td>Terminal. Default font + window size. 1920x1080 capture.<br><img src="https://raw.githubusercontent.com/fleetdm/fleet/main/handbook/marketing/images/terminal-default-font-size-1920x1080-capture.png" width="800" alt="terminal-default-font-size-1920x1080capture.png"></td></tr>
<tr><td>Terminal. Increased font+window size. 1920x1080 capture.<br><img src="https://raw.githubusercontent.com/fleetdm/fleet/main/handbook/marketing/images/terminal-increased-font-size-1920x1080-capture.png" width="800" alt="terminal-increased-font-size-1920x1080capture.png"></td></tr>
</table>

<table>
<tr><td>Browser default content + window size. 1920x1080 capture.<br><img src="https://raw.githubusercontent.com/fleetdm/fleet/main/handbook/marketing/images/browser-default-window-size-1920x1080-capture.png" width="800" alt="browser-default-window-size-1920x1080-capture.png"></td></tr>
<tr><td>Browser increased content + window size. 1920x1080 capture.<br><img src="https://raw.githubusercontent.com/fleetdm/fleet/main/handbook/marketing/images/browser-increased-window-size-1920x1080-capture.png" width="800" alt="browser-increased-window-size-1920x1080-capture.png"></td></tr>
</table>

</details>


## Before you record

- Resize the window for legibility - Resize the application window to 1920x1080 before recording. Avoid recording across an ultrawide display or a full multi-monitor desktop. Most viewers watch in a player smaller than full-screen, often on a laptop. UI elements that look fine on your screen will be unreadable once the video is scaled to 1080p for playback.
- Zoom in - Increase font size in your terminal, IDE, or browser beyond your daily working setup. Text should be legible to a viewer watching the published video in a small window.
- Use a clean browser profile - Close unnecessary tabs and extensions.
- Pause notifications - Close or minimize anything that might expose private data. Hide desktop icons or confirm no filenames contain sensitive info.
- If you are recording talking heads (webinar, conversation, explainer video, round table, etc) then Zoom is preferred with the following settings to give best video output to work with in post production:
<details style="background: #f6f8fa; border: 1px solid #d0d7de; border-radius: 6px; padding: 8px 16px;">
<summary style="cursor: pointer; font-weight: 600;">Zoom settings</summary>

When recording using Zoom, set your recording options to record speakers in a separate stream from the shared content, so we can overlay headshots while maximizing the shared content.

<img src="https://raw.githubusercontent.com/fleetdm/fleet/main/handbook/marketing/images/recording-zoom-settings-01.png" width="800" alt="recording-zoom-settings-01.png"> 

Also, no timestamps or names displayed, and optimize for 3rd party video editor:

<img src="https://raw.githubusercontent.com/fleetdm/fleet/main/handbook/marketing/images/recording-zoom-settings-02.png" width="800" alt="recording-zoom-settings-02.png">

</details>

## Capture settings

- Capture resolution - Set your recorder's output resolution to 1920x1080 minimum. Higher (e.g. 3840x2160) is fine if your tool supports it — it gives editors more detail to zoom and reframe with.
- Frame rate - 30 fps.

Remember: capture resolution does not change how the recorded UI looks at 1080p. That's still determined by your window size.

## File output

- Format - Save as `.mp4` (h.264/AAC) or `.mov`. Don't use `.webm`.
- Filenames
   - Sprint demos: `[release number]-[issue #]-demo-[YYYY-MM-DD].[extension]`
   - Webinars: `[webinar title]-webinar-[YYYY-MM-DD].[extension]`
- Sharing - Make sure videos are shared as downloadable. Prefer sharing through Google Drive. If we have to pull a video from YouTube, it has already lost a generation of quality and will look worse in the final video.

## While recording

- Add buffer silence - Leave a few seconds of silence at the start and end of each recording. This gives editors room to work with.
- Move the mouse deliberately
   - Pause briefly on important UI elements.
   - Wait for results to load before moving on.
   - Don't wave your mouse around or circle what you're talking about. It makes edit cuts look worse and reduces editing flexibility.
   - Don't scroll up and down needlessly. If we have to hide private information in post, scrolling makes it about 4x harder and messier.
- Don't add captions over videos - They're added later when needed.
   - Don't enable closed captions when playing your video for a sprint demo recording.
   - Don't have your recording platform (e.g. Zoom) add names, positions, etc. over speaker headshot videos.

## Privacy

While recording, protect any private data that should not be shown publicly.

### What to protect

- Public IP addresses
   - Non-routable nets ([RFC 1918](https://datatracker.ietf.org/doc/html/rfc1918)) are OK:
      - 10.0.0.0 – 10.255.255.255 (10/8 prefix)
      - 172.16.0.0 – 172.31.255.255 (172.16/12 prefix)
      - 192.168.0.0 – 192.168.255.255 (192.168/16 prefix)
- Real email addresses
- Passwords
- AWS, API, and other tokens
- Login credentials (e.g. Microsoft Entra ID)
- Apple IDs
- Apple serial numbers for real hardware
- Publicly accessible server names, even if not live after the demo (editors won't know which are safe). Often appear in address bars and MDM server fields as `*.ngrok.app`, `dogfood`, etc.
- UUIDs
   - Derived from hardware (e.g. MDM-issued host identifiers)
   - API response user IDs
   - Tenant or org IDs
- Customer names
- Real phone numbers
- Physical addresses

### How to protect it

- Don't rely on blur or pixelation. Open-source tools can recover blurred and pixelated text, especially monospace text from terminals and IDEs. Use one of these approaches instead:
   - Best: Don't show the data. Use placeholder values, demo accounts, or dummy data before you start recording.
   - Acceptable: Cover the data with an opaque rectangle (solid black or matching the surrounding UI) in post-production. Confirm the rectangle fully covers the data on every frame, including motion.

### Obscuro

[Obscuro](https://chromewebstore.google.com/detail/obscuro-sensitive-data-hi/peljfjmphjkflheafjlnjmkmdppbcjap?hl=en-US) is a Chrome browser extension that hides sensitive data dynamically using regex patterns and CSS selectors. It supports a shared configuration file, so you don't have to redefine what to hide each time, and everyone can contribute to it. Use its "replace text" feature rather than its blur feature, since blur can be undone.

> Obscuro is from a legitimate security company ([Intezer](https://intezer.com/))and passed a security check by Claude Opus, but it has not yet been validated by Fleet's security team. Until it has, run it in a disposable VM or a separate macOS user account, and delete the environment/account afterward.

Below is a shared config that currently auto-redacts the most common data we have to hide in post-production of sprint-demo videos. It does not yet cover everything in the list above.

```json
{
  "ignore": {
    "regex": [
      {
        "flags": "i",
        "pattern": "support@mycompany\\.com"
      },
      {
        "flags": "i",
        "pattern": "sales@mycompany\\.com"
      },
      {
        "flags": "i",
        "pattern": "demo[0-9]+@example\\.com"
      },
      {
        "flags": "i",
        "pattern": "test@test\\.com"
      }
    ],
    "selectors": [
      ".public-data",
      "#marketing-section *",
      ".footer-contact",
      "input[name='support_email']",
      "[data-public='true']"
    ]
  },
  "regex": [
    {
      "flags": "g",
      "pattern": "\\b\\d{3}-\\d{2}-\\d{4}\\b"
    },
    {
      "flags": "g",
      "pattern": "\\b\\d{4}[\\s-]?\\d{4}[\\s-]?\\d{4}[\\s-]?\\d{4}\\b"
    },
    {
      "flags": "gi",
      "pattern": "[A-Z0-9._%+-]+@[A-Z0-9.-]+\\.[A-Z]{2,}"
    },
    {
      "flags": "g",
      "pattern": "\\b\\d{3}[-.\\s]?\\d{3}[-.\\s]?\\d{4}\\b"
    },
    {
      "flags": "g",
      "pattern": "\\b(?=[A-Z0-9]*[A-Z])(?=[A-Z0-9]*[0-9])[A-Z0-9]{10,12}\\b"
    },
    {
      "flags": "g",
      "pattern": "\\b(?!10\\.)(?!172\\.(?:1[6-9]|2[0-9]|3[01])\\.)(?!192\\.168\\.)(?!127\\.)(?!0\\.)(?!169\\.254\\.)(?!2[4-5][0-9]\\.)(?:(?:25[0-5]|2[0-4]\\d|1?\\d{1,2})\\.){3}(?:25[0-5]|2[0-4]\\d|1?\\d{1,2})\\b"
    },
    {
      "flags": "gi",
      "pattern": "\\b[a-z0-9-]+\\.ngrok\\.app\\b"
    },
    {
      "flags": "gi",
      "pattern": "\\b[a-z0-9-]+\\.fleetdm\\.com\\b"
    },
    {
      "flags": "g",
      "pattern": "AKIA[0-9A-Z]{16}"
    },
    {
      "flags": "g",
      "pattern": "(?<![A-Za-z0-9/+=])[A-Za-z0-9/+=]{40}(?![A-Za-z0-9/+=])"
    },
    {
      "flags": "g",
      "pattern": "(AKIA|ASIA|AROA|AIDA|ANPA|ANVA|APKA)[0-9A-Z]{16}"
    },
    {
      "flags": "g",
      "pattern": "sk_live_[0-9a-zA-Z]{24}"
    },
    {
      "flags": "g",
      "pattern": "sk_test_[0-9a-zA-Z]{24}"
    },
    {
      "flags": "g",
      "pattern": "sk-[a-zA-Z0-9]{48}"
    },
    {
      "flags": "g",
      "pattern": "ghp_[A-Za-z0-9_]{36}"
    },
    {
      "flags": "g",
      "pattern": "github_pat_[A-Za-z0-9]{22}_[A-Za-z0-9]{59}"
    },
    {
      "flags": "g",
      "pattern": "AIza[0-9A-Za-z\\-_]{35}"
    },
    {
      "flags": "g",
      "pattern": "\\beyJ[a-zA-Z0-9\\-_]+\\.[a-zA-Z0-9\\-_]+\\.[a-zA-Z0-9\\-_]+"
    },
    {
      "flags": "g",
      "pattern": "(?i)(api[-_]?key|access[-_]?token|auth[-_]?token|secret[-_]?key|bearer)\\s*[:=]\\s*['\"]?[A-Za-z0-9\\-_=]{20,}"
    }
  ],
  "selectors": [
    "[data-sensitive='true']",
    ".customer-email",
    ".customer-phone",
    ".customer-address",
    ".ssn",
    ".credit-card",
    ".account-number",
    "input[name='email']",
    "input[name='phone']",
    "input[type='tel']",
    ".contact-info",
    "[data-pii='true']"
  ],
  "version": "1"
}
```

<meta name="maintainedBy" value="danbgordon">
