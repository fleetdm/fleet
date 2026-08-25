import React from "react";
import { QRCodeSVG } from "qrcode.react";

// Pinned to dark-on-light in both themes rather than using the equivalent
// design tokens, which invert under `body.dark-mode` ($ui-fleet-black-75
// becomes #bebebf, $ui-off-white becomes #1e2128). Many scanners reject a
// light-on-dark QR code.
const FOREGROUND_COLOR = "#515774";
const BACKGROUND_COLOR = "#f9fafc";

const SIZE = 148;

const baseClass = "enroll-qr-code";

interface IEnrollQrCodeProps {
  url: string;
}

// The quiet zone comes from the frame's padding, which is the same light color
// as the code's background, so no `marginSize` is rendered inside the SVG.
const EnrollQrCode = ({ url }: IEnrollQrCodeProps) => (
  <div className={`${baseClass} form-field`}>
    <span className="form-field__label">To test, scan the QR code:</span>
    <div className={`${baseClass}__frame`} data-testid="enroll-qr-code">
      <QRCodeSVG
        value={url}
        size={SIZE}
        level="M"
        fgColor={FOREGROUND_COLOR}
        bgColor={BACKGROUND_COLOR}
        title="Enrollment link QR code"
      />
    </div>
  </div>
);

export default EnrollQrCode;
