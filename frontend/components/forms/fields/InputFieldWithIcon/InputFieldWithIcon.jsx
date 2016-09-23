import React, { Component, PropTypes } from 'react';
import radium from 'radium';
import componentStyles from './styles';


class InputFieldWithIcon extends Component {
  static propTypes = {
    autofocus: PropTypes.bool,
    error: PropTypes.string,
    iconName: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func,
    placeholder: PropTypes.string,
    style: PropTypes.object,
    type: PropTypes.string,
  };

  static defaultProps = {
    autofocus: false,
    style: {},
    type: 'text',
  };

  constructor (props) {
    super(props);
    this.state = { value: null };
  }

  componentDidMount () {
    const { autofocus } = this.props;
    const { input } = this;

    if (autofocus) input.focus();

    return false;
  }

  onInputChange = (evt) => {
    evt.preventDefault();

    const { value } = evt.target;
    const { onChange } = this.props;

    this.setState({ value });
    return onChange(evt);
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
        />
        <i className={iconName} style={[iconStyles(value), iconErrorStyles(error), style]} />
      </div>
    );
  }
}

export default radium(InputFieldWithIcon);
