import React, { Component } from 'react';
import classnames from 'classnames';

import base, { basePropTypes } from '../base';

const baseClass = 'lock-svg';

class Lock extends Component {
  static propTypes = {
    ...basePropTypes,
  };

  render () {
    const { alt, style, variant, className } = this.props;

    const lockBorderClasses = classnames(
      `${baseClass}__border`,
      `${baseClass}__border--${variant}`
    );

    const lockFill1Classes = classnames(
      `${baseClass}__fill1`,
      `${baseClass}__fill1--${variant}`
    );

    const lockFill2Classes = classnames(
      `${baseClass}__fill2`,
      `${baseClass}__fill2--${variant}`
    );

    return (
      <svg
        width="12px"
        height="15px"
        viewBox="0 0 12 15"
        version="1.1"
        xmlns="http://www.w3.org/2000/svg"
        xmlnsXlink="http://www.w3.org/1999/xlink"
        alt={alt}
        style={style}
        className={className}
      >
        <g stroke="none" strokeWidth="1" fill="none" fillRule="evenodd">
          <g transform="translate(-671.000000, -594.000000)" className={lockBorderClasses}>
            <g transform="translate(272.000000, 340.000000)">
              <g transform="translate(41.000000, 170.000000)">
                <g transform="translate(359.000000, 84.500000)">
                  <path d="M0.788040967,9.60172925 C0.788040967,6.76454864 0.38717522,0.182911373 5.07553236,0.182911373 C9.76388951,0.182911373 9.36302401,6.68931331 9.363024,9.51568956" />
                  <circle id="Oval-2" className={lockFill1Classes} cx="5" cy="9" r="5" />
                  <circle id="Oval-2" className={lockFill2Classes} cx="5" cy="9" r="1" />
                </g>
              </g>
            </g>
          </g>
        </g>
      </svg>
    );
  }
}

export default base(Lock);
