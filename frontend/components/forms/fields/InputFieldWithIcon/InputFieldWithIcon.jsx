import React, { Component, PropTypes } from 'react';
import radium from 'radium';
import Icon from '../../../icons/Icon';
import componentStyles from './styles';

class InputFieldWithIcon extends Component {
  static propTypes = {
    error: PropTypes.string,
    iconName: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func,
    placeholder: PropTypes.string,
    style: PropTypes.object,
    type: PropTypes.string,
  };

  static defaultProps = {
    style: {},
    type: 'text',
  };

  constructor (props) {
    super(props);
    this.state = { value: null };
  }

  onInputChange = (evt) => {
    evt.preventDefault();

    const { value } = evt.target;
    const { onChange } = this.props;

    this.setState({ value });
    return onChange(evt);
  }

  iconVariant = () => {
    const { error } = this.props;
    const { value } = this.state;

    if (error) return 'error';

    if (value) return 'colored';

    return 'default';
  }

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
    const { containerStyles, iconStyles, inputErrorStyles, inputStyles } = componentStyles;
    const { value } = this.state;
    const { iconVariant, onInputChange } = this;

    return (
      <div style={containerStyles}>
        {this.renderHeading()}
        <input
          name={name}
          onChange={onInputChange}
          placeholder={placeholder}
          style={[inputStyles(value), inputErrorStyles(error), style]}
          type={type}
        />
        <Icon name={iconName} style={iconStyles} variant={iconVariant()} />
      </div>
    );
  }
}

export default radium(InputFieldWithIcon);
