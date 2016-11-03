import React, { Component } from 'react';

import base, { basePropTypes } from '../base';
import color from '../../../../styles/colors';

class Envelope extends Component {
  static propTypes = {
    ...basePropTypes,
  };

  static variants = {
    default: {
      clipboardFill: '#858495',
      paperFill: '#FBFBFD',
    },
    copied: {
      clipboardFill: color.brandLight,
      paperFill: '#FBFBFD',
    },
  };

  render () {
    const { alt, onClick, style, variant, className } = this.props;
    const iconVariant = Envelope.variants[variant];
    const { clipboardFill, paperFill } = iconVariant;

    return (
      <svg
        width="25px"
        height="26px"
        onClick={onClick}
        viewBox="0 0 25 26"
        alt={alt}
        style={style}
        className={className}
      >
        <g stroke="none" strokeWidth="1" fill="none" fillRule="evenodd">
          <g transform="translate(-1037.000000, -132.000000)">
            <g transform="translate(290.000000, 30.000000)">
              <g transform="translate(747.000000, 102.000000)">
                <g transform="translate(0.106758, 0.000000)">
                  <rect fill={clipboardFill} x="0" y="0" width="17.9764936" height="23" rx="1" />
                  <rect fill={paperFill} x="2.99608227" y="2" width="11.9843291" height="2" />
                </g>
                <g transform="translate(9.095005, 6.000000)">
                  <path d="M1.42670584,0 C0.638960403,0 0,0.639795918 0,1.42857143 L0,18.5714286 C0,19.3602041 0.638960403,20 1.42670584,20 L13.5537055,20 C14.341451,20 14.9804114,19.3602041 14.9804114,18.5714286 L14.9804114,5.71428571 L9.27358799,0 L1.42670584,0 Z" id="Combined-Shape" fill={paperFill} />
                  <path d="M9.27358799,5.71428571 L9.27358799,2.02040816 L12.9626417,5.71428571 L9.27358799,5.71428571 Z M13.5537055,18.5714286 L1.42670584,18.5714286 L1.42670584,1.42857143 L7.84688214,1.42857143 L7.84688214,7.14285714 L13.5537055,7.14285714 L13.5537055,18.5714286 Z M1.42670584,0 C0.638960403,0 0,0.639795918 0,1.42857143 L0,18.5714286 C0,19.3602041 0.638960403,20 1.42670584,20 L13.5537055,20 C14.341451,20 14.9804114,19.3602041 14.9804114,18.5714286 L14.9804114,5.71428571 L9.27358799,0 L1.42670584,0 Z" id="Icon" fill={clipboardFill} />
                </g>
              </g>
            </g>
          </g>
        </g>
      </svg>
    );
  }
}

export default base(Envelope);
