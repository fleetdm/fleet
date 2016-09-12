import React, { Component } from 'react';
import base, { basePropTypes } from '../base';
import color from '../../../../styles/color';

class KolideLogo extends Component {
  static propTypes = {
    ...basePropTypes,
  };

  static variants = {
    default: {
      logo: color.lightGrey,
      logoFill: color.darkGrey,
    },
  };

  render () {
    const { alt, style, variant } = this.props;
    const fill = KolideLogo.variants[variant];

    return (
      <svg
        width="40px"
        height="41px"
        viewBox="0 0 40 41"
        version="1.1"
        xmlns="http://www.w3.org/2000/svg"
        xmlnsXlink="http://www.w3.org/1999/xlink"
        alt={alt}
        style={style}
      >
        <g stroke="none" strokeWidth="1" fill="none" fillRule="evenodd">
          <g transform="translate(-434.000000, -1083.000000)">
            <g transform="translate(0.000000, 1065.500000)">
              <g transform="translate(434.000000, 18.000000)">
                <path d="M33.25,4.5975082 C33.0240516,3.74168852 32.3545434,3.07455738 31.4997073,2.84447541 L21.3483959,0.125295082 C20.4927401,-0.103557377 19.5800352,0.14095082 18.9538057,0.766196721 C18.9538057,0.766196721 12.9144843,6.80532761 9.89482359,9.82489305 L0.765772915,18.9536557 C0.139625374,19.580459 -0.104800856,20.4933279 0.124707341,21.3490656 L2.84478931,31.5009508 C3.0740516,32.3563607 3.74200242,33.0239836 4.59757619,33.252918 L29.4420844,39.910623 C30.2971664,40.1395574 31.210609,39.8951311 31.8367565,39.2682459 L39.2687237,31.8369344 C39.8948713,31.2112787 40.1392975,30.2985738 39.9108549,29.4431639 L33.25,4.5975082 Z" fill={fill.logo} />
                <polygon fill={fill.logoFill} points="32.5193279 32.5449508 26.0222787 32.5449508 21.0085902 24.4649508 17.3807213 24.4679836 20.3447377 26.0453607 20.3551475 32.5449508 14.7725246 32.5449508 14.751623 17.1566721 11.1396557 15.2310164 11.1368689 13.1216721 20.3288361 13.1216721 20.3376885 20.0761803 21.6184262 20.0761803 26.0222787 13.1216721 32.5193279 13.1216721 26.5808852 22.8397869" />
              </g>
            </g>
          </g>
        </g>
      </svg>
    );
  }
}

export default base(KolideLogo);
