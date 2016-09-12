import React, { Component, PropTypes } from 'react';
import { keys, pick } from 'lodash';
import radium from 'radium';

export const basePropTypes = {
  alt: PropTypes.string,
  name: PropTypes.string,
  style: PropTypes.object,
  variant: PropTypes.string,
};

export default function (SVGComponent) {
  class ComponentWrapper extends Component {
    static propTypes = {
      ...basePropTypes,
    };

    static defaultProps = {
      variant: 'default',
    };

    render () {
      const svgProps = pick(this.props, keys(ComponentWrapper.propTypes));
      return <SVGComponent {...svgProps} />;
    }
  }

  return radium(ComponentWrapper);
}
