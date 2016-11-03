import React, { Component, PropTypes } from 'react';
import { keys, noop, pick } from 'lodash';

export const basePropTypes = {
  alt: PropTypes.string,
  name: PropTypes.string,
  onClick: PropTypes.func,
  style: PropTypes.object,
  variant: PropTypes.string,
  className: PropTypes.string,
};

export default function (SVGComponent) {
  class ComponentWrapper extends Component {
    static propTypes = {
      ...basePropTypes,
    };

    static defaultProps = {
      onClick: noop,
      variant: 'default',
      className: '',
    };

    render () {
      const svgProps = pick(this.props, keys(ComponentWrapper.propTypes));
      return <SVGComponent {...svgProps} />;
    }
  }

  return ComponentWrapper;
}
