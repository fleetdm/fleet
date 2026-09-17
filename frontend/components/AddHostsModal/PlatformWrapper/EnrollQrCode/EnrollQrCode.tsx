import React from "react";
import { QRCodeSVG } from "qrcode.react";

// Pinned to dark-on-light in both themes rather than using the equivalent
// design tokens, which invert under `body.dark-mode` ($ui-fleet-black-75
// becomes #bebebf, $ui-off-white becomes #1e2128). Many scanners reject a
// light-on-dark QR code. These are the same literals the OTA enrollment page
// hardcodes for the same reason (frontend/templates/enroll-ota.html).
const FOREGROUND_COLOR = "#515774";
const BACKGROUND_COLOR = "#f9fafc";

// Matches the OTA enrollment page. Enroll secrets can be up to 255 characters,
// and module size shrinks as the encoded URL grows, so this is sized to keep
// the code scannable rather than to the smallest size that looks right.
const SIZE = 208;

// The spec's 4-module quiet zone, emitted inside the SVG so it scales with
// module size. It cannot come from the frame's padding, which is a fixed pixel
// value and would fall short of 4 modules for short URLs.
const MARGIN_MODULES = 4;

const baseClass = "enroll-qr-code";

interface IEnrollQrCodeProps {
  url: string;
}

const EnrollQrCode = ({ url }: IEnrollQrCodeProps) => (
  <div className={`${baseClass} form-field`}>
    <div className="form-field__label">To test, scan the QR code:</div>
    <div className={`${baseClass}__frame`} data-testid="enroll-qr-code">
      <QRCodeSVG
        value={url}
        size={SIZE}
        level="M"
        marginSize={MARGIN_MODULES}
        fgColor={FOREGROUND_COLOR}
        bgColor={BACKGROUND_COLOR}
        title="Enrollment link QR code"
      />
    </div>
  </div>
);

export default EnrollQrCode;
