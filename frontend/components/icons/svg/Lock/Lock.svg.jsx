import React, { Component } from 'react';
import base, { basePropTypes } from '../base';

class Lock extends Component {
  static propTypes = {
    ...basePropTypes,
  };

  static variants = {
    default: (
      <g stroke="none" strokeWidth="1" fill="none" fillRule="evenodd">
        <g transform="translate(-671.000000, -594.000000)" stroke="#B9C2E4">
          <g transform="translate(272.000000, 340.000000)">
            <g transform="translate(41.000000, 170.000000)">
              <g transform="translate(359.000000, 84.500000)">
                <path d="M0.788040967,9.60172925 C0.788040967,6.76454864 0.38717522,0.182911373 5.07553236,0.182911373 C9.76388951,0.182911373 9.36302401,6.68931331 9.363024,9.51568956" />
                <circle id="Oval-2" fill="#EAEEFB" cx="5" cy="9" r="5" />
                <circle id="Oval-2" fill="#B9C2E4" cx="5" cy="9" r="1" />
              </g>
            </g>
          </g>
        </g>
      </g>
    ),
    colored: (
      <g stroke="none" strokeWidth="1" fill="none" fillRule="evenodd">
        <g transform="translate(-671.000000, -594.000000)" stroke="#C48DED">
          <g transform="translate(272.000000, 340.000000)">
            <g transform="translate(41.000000, 149.000000)">
              <g transform="translate(359.000000, 105.500000)">
                <path d="M0.788040967,9.60172925 C0.788040967,6.76454864 0.38717522,0.182911373 5.07553236,0.182911373 C9.76388951,0.182911373 9.36302401,6.68931331 9.363024,9.51568956" />
                <circle fill="#EED6FF" cx="5" cy="9" r="5" />
                <circle fill="#C48DED" cx="5" cy="9" r="1" />
              </g>
            </g>
          </g>
        </g>
      </g>
    ),
  };

  render () {
    const { alt, style, variant } = this.props;

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
      >
        {Lock.variants[variant]}
      </svg>
    );
  }
}

export default base(Lock);
