import React, { Component, PropTypes } from 'react';
import radium from 'radium';

import componentStyles from './styles';

class Button extends Component {
  static propTypes = {
    onClick: PropTypes.func,
    style: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    text: PropTypes.string,
    type: PropTypes.string,
    variant: PropTypes.string,
  };

  static defaultProps = {
    style: {},
    variant: 'default',
  };

  render () {
    const { onClick, style, text, type, variant } = this.props;

    return (
      <button
        onClick={onClick}
        style={[componentStyles[variant], style]}
        type={type}
      >
        {text}
      </button>
    );
  }
}

export default radium(Button);
