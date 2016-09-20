import React, { Component, PropTypes } from 'react';
import radium from 'radium';
import componentStyles from './styles';

class GradientButton extends Component {
  static propTypes = {
    onClick: PropTypes.func,
    style: PropTypes.object,
    text: PropTypes.string,
    type: PropTypes.string,
  };

  static defaultProps = {
    style: {},
  };

  render () {
    const { onClick, style, text, type } = this.props;

    return (
      <button
        onClick={onClick}
        style={[componentStyles, style]}
        type={type}
      >
        {text}
      </button>
    );
  }
}

export default radium(GradientButton);
