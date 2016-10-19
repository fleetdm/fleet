import React, { Component } from 'react';

import base, { basePropTypes } from '../base';

class Check extends Component {
  static propTypes = {
    ...basePropTypes,
  };

  static variants = {
    default: (
      <g stroke="none" strokeWidth="1" fill="none" fillRule="evenodd">
        <g transform="translate(-452.000000, -373.000000)">
          <g transform="translate(310.000000, 226.000000)">
            <g transform="translate(0.000000, 83.500000)">
              <g transform="translate(192.000000, 114.000000) rotate(-315.000000) translate(-192.000000, -114.000000) translate(142.000000, 64.000000)">
                <circle fill="#4ED061" cx="49.7056816" cy="49.7056816" r="49.2991274" />
                <path d="M68.850926,68.2514103 L68.8702874,68.2707717 L74.3744581,62.7666011 L34.0105401,22.4026832 L28.5063695,27.9068538 L63.3467554,62.7472397 L48.7968715,77.2971236 L54.3010421,82.8012943 L68.850926,68.2514103 Z" fill="#FFFFFF" transform="translate(51.440414, 52.601989) rotate(45.000000) translate(-51.440414, -52.601989) " />
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
        width="100px"
        height="100px"
        viewBox="0 0 100 100"
        alt={alt}
        style={style}
      >
        {Check.variants[variant]}
      </svg>
    );
  }
}

export default base(Check);
