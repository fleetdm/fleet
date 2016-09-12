import React, { Component, PropTypes } from 'react';
import radium from 'radium';
import Icon from '../../../icons/Icon';
import componentStyles from './styles';

class InputFieldWithIcon extends Component {
  static propTypes = {
    iconName: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func,
    placeholder: PropTypes.string,
    type: PropTypes.string,
  };

  static defaultProps = {
    type: 'text',
  };

  constructor (props) {
    super(props);
    this.state = { value: null };
  }

  onInputChange = (evt) => {
    const { value } = evt.target;
    const { onChange } = this.props;

    this.setState({ value });
    return onChange(evt);
  };

  render () {
    const { iconName, name, placeholder, type } = this.props;
    const { containerStyles, iconStyles, inputStyles, placeholderStyles } = componentStyles;
    const { value } = this.state;
    const { onInputChange } = this;
    const iconVariant = value ? 'colored' : 'default';

    return (
      <div style={containerStyles}>
        <div style={placeholderStyles(value)}>{placeholder}</div>
        <input
          name={name}
          onChange={onInputChange}
          placeholder={placeholder}
          style={inputStyles(value)}
          type={type}
        />
        <Icon name={iconName} style={iconStyles} variant={iconVariant} />
      </div>
    );
  }
}

export default radium(InputFieldWithIcon);
