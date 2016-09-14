import React, { Component, PropTypes } from 'react';
import radium from 'radium';
import componentStyles from './styles';

class GradientButton extends Component {
  static propTypes = {
    disabled: PropTypes.bool,
    onClick: PropTypes.func,
    style: PropTypes.object,
    text: PropTypes.string,
    type: PropTypes.string,
  };

  static defaultProps = {
    style: {},
  };

  render () {
    const { disabled, onClick, style, text, type } = this.props;

    return (
      <button
        disabled={disabled}
        onClick={onClick}
        style={[componentStyles(disabled), style]}
        type={type}
      >
        {text}
      </button>
    );
  }
}

export default radium(GradientButton);
