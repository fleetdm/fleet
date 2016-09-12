import React, { Component } from 'react';
import base, { basePropTypes } from '../base';

export class User extends Component {
  static propTypes = {
    ...basePropTypes,
  };

  static variants = {
    default: (
      <g stroke="none" strokeWidth="1" fill="none" fillRule="evenodd">
        <g transform="translate(-672.000000, -519.000000)" fill="#EAEEFB" stroke="#B9C2E4">
          <g transform="translate(272.000000, 340.000000)">
            <g transform="translate(41.000000, 170.000000)">
              <g transform="translate(360.000000, 9.500000)">
                <path d="M5,12 C7.76142375,12 10,10.4198285 10,8.47058824 C10,6.52134794 7.76142375,4.94117647 5,4.94117647 C2.23857625,4.94117647 0,6.52134794 0,8.47058824 C0,10.4198285 2.23857625,12 5,12 Z" />
                <ellipse cx="5" cy="3.52941176" rx="3.57142857" ry="3.52941176" />
              </g>
            </g>
          </g>
        </g>
      </g>
    ),
    circle: (
      <g>
        <defs>
          <circle id="path-1" cx="48" cy="48" r="48" />
          <mask id="mask-2" maskContentUnits="userSpaceOnUse" maskUnits="objectBoundingBox" x="-1" y="-1" width="98" height="98">
            <rect x="-1" y="-1" width="98" height="98" fill="white" />
            <use xlinkHref="#path-1" fill="black" />
          </mask>
          <mask id="mask-4" maskContentUnits="userSpaceOnUse" maskUnits="objectBoundingBox" x="-1" y="-1" width="98" height="98">
            <rect x="-1" y="-1" width="98" height="98" fill="white" />
            <use xlinkHref="#path-1" fill="black" />
          </mask>
          <circle id="path-5" cx="48" cy="39.6" r="24" />
          <mask id="mask-6" maskContentUnits="userSpaceOnUse" maskUnits="objectBoundingBox" x="0" y="0" width="48" height="48" fill="white">
            <use xlinkHref="#path-5" />
          </mask>
        </defs>
        <g id="Page-1" stroke="none" strokeWidth="1" fill="none" fillRule="evenodd">
          <g id="Kolide-App-Login-Base-State" transform="translate(-453.000000, -374.000000)">
            <g id="Page-Content">
              <g id="Child-Div" transform="translate(272.000000, 340.000000)">
                <g id="Oval-+-Oval-Mask" transform="translate(182.000000, 35.500000)">
                  <mask id="mask-3" fill="white">
                    <use xlinkHref="#path-1" />
                  </mask>
                  <g id="Mask" stroke="#B9C2E4" mask="url(#mask-2)" strokeWidth="2">
                    <use mask="url(#mask-4)" xlinkHref="#path-1" />
                  </g>
                  <path d="M48.2666667,103.6 C68.2961936,103.6 84.5333333,92.1384896 84.5333333,78 C84.5333333,63.8615104 68.2961936,52.4 48.2666667,52.4 C28.2371397,52.4 12,63.8615104 12,78 C12,92.1384896 28.2371397,103.6 48.2666667,103.6 Z" id="Oval" stroke="#D2DAF4" strokeWidth="3" fill="#EAEEFB" mask="url(#mask-3)" />
                  <g id="Oval" mask="url(#mask-3)" strokeWidth="6" stroke="#D2DAF4" fill="#EAEEFB">
                    <use mask="url(#mask-6)" xlinkHref="#path-5" />
                  </g>
                </g>
              </g>
            </g>
          </g>
        </g>
      </g>
    ),
    colored: (
      <g stroke="none" strokeWidth="1" fill="none" fillRule="evenodd">
        <g transform="translate(-672.000000, -519.000000)" fill="#EED6FF" stroke="#C48DED">
          <g transform="translate(272.000000, 340.000000)">
            <g transform="translate(41.000000, 149.000000)">
              <g transform="translate(360.000000, 30.500000)">
                <path d="M5,12 C7.76142375,12 10,10.4198285 10,8.47058824 C10,6.52134794 7.76142375,4.94117647 5,4.94117647 C2.23857625,4.94117647 0,6.52134794 0,8.47058824 C0,10.4198285 2.23857625,12 5,12 Z" />
                <ellipse cx="5" cy="3.52941176" rx="3.57142857" ry="3.52941176" />
              </g>
            </g>
          </g>
        </g>
      </g>
    ),
  };

  render () {
    const { alt, style, variant } = this.props;

    if (variant === 'circle') {
      return (
        <svg
          width="98px"
          height="99px"
          viewBox="0 0 98 99"
          version="1.1"
          xmlns="http://www.w3.org/2000/svg"
          xmlnsXlink="http://www.w3.org/1999/xlink"
          alt={alt}
          style={style}
        >
          {User.variants[variant]}
        </svg>
      );
    }

    return (
      <svg
        width="12px"
        height="13px"
        viewBox="0 0 12 13"
        version="1.1"
        xmlns="http://www.w3.org/2000/svg"
        xmlnsXlink="http://www.w3.org/1999/xlink"
        alt={alt}
        style={style}
      >
        {User.variants[variant]}
      </svg>
    );
  }
}

export default base(User);
