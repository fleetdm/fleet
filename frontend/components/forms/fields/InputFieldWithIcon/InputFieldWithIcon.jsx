import React, { PropTypes } from 'react';
import radium from 'radium';

import componentStyles from './styles';
import InputField from '../InputField';

class InputFieldWithIcon extends InputField {
  static propTypes = {
    autofocus: PropTypes.bool,
    defaultValue: PropTypes.string,
    error: PropTypes.string,
    iconName: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func,
    placeholder: PropTypes.string,
    style: PropTypes.object,
    type: PropTypes.string,
  };

  renderHeading = () => {
    const { error, placeholder } = this.props;
    const { value } = this.state;
    const { errorStyles, placeholderStyles } = componentStyles;

    if (error) {
      return <div style={errorStyles}>{error}</div>;
    }

    return <div style={placeholderStyles(value)}>{placeholder}</div>;
  }

  render () {
    const { error, iconName, name, placeholder, style, type } = this.props;
    const { containerStyles, iconStyles, iconErrorStyles, inputErrorStyles, inputStyles } = componentStyles;
    const { value } = this.state;
    const { onInputChange } = this;

    return (
      <div style={containerStyles}>
        {this.renderHeading()}
        <input
          name={name}
          onChange={onInputChange}
          className="input-with-icon"
          placeholder={placeholder}
          ref={(r) => { this.input = r; }}
          style={[inputStyles(value, type), inputErrorStyles(error), style]}
          type={type}
          value={value}
        />
        {iconName && <i className={iconName} style={[iconStyles(value), iconErrorStyles(error), style]} />}
      </div>
    );
  }
}

export default radium(InputFieldWithIcon);
