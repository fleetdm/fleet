import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';
import radium from 'radium';

import componentStyles from './styles';

class InputField extends Component {
  static propTypes = {
    autofocus: PropTypes.bool,
    defaultValue: PropTypes.string,
    error: PropTypes.string,
    inputClassName: PropTypes.string, // eslint-disable-line react/forbid-prop-types
    inputWrapperStyles: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    inputOptions: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    label: PropTypes.string,
    labelClassName: PropTypes.string,
    labelStyles: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    name: PropTypes.string,
    onChange: PropTypes.func,
    placeholder: PropTypes.string,
    style: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    type: PropTypes.string,
  };

  static defaultProps = {
    autofocus: false,
    defaultValue: '',
    inputWrapperStyles: {},
    inputOptions: {},
    label: null,
    labelStyles: {},
    style: {},
    type: 'text',
  };

  constructor (props) {
    super(props);

    const { defaultValue } = props;

    this.state = { value: defaultValue };
  }

  componentDidMount () {
    const { autofocus } = this.props;
    const { input } = this;

    if (autofocus) {
      input.focus();
    }

    return false;
  }

  onInputChange = (evt) => {
    evt.preventDefault();

    const { value } = evt.target;
    const { onChange } = this.props;

    this.setState({ value });
    return onChange(evt);
  }

  renderLabel = () => {
    const { componentLabelStyles } = componentStyles;
    const { error, label, labelClassName, labelStyles, name } = this.props;

    if (!label) {
      return false;
    }

    return (
      <label
        className={labelClassName}
        htmlFor={name}
        style={[componentLabelStyles(error), labelStyles]}
      >
        {error || label}
      </label>
    );
  }

  render () {
    const { error, inputClassName, inputOptions, inputWrapperStyles, name, placeholder, style, type } = this.props;
    const { inputErrorStyles, inputStyles } = componentStyles;
    const { value } = this.state;
    const { onInputChange, renderLabel } = this;
    const className = classnames('input-with-icon', inputClassName);

    if (type === 'textarea') {
      return (
        <div style={inputWrapperStyles}>
          {renderLabel()}
          <textarea
            name={name}
            onChange={onInputChange}
            className={className}
            placeholder={placeholder}
            ref={(r) => { this.input = r; }}
            style={[inputStyles(type, value), inputErrorStyles(error), style]}
            type={type}
            {...inputOptions}
            value={value}
          />
        </div>
      );
    }

    return (
      <div style={inputWrapperStyles}>
        {renderLabel()}
        <input
          className={className}
          name={name}
          onChange={onInputChange}
          placeholder={placeholder}
          ref={(r) => { this.input = r; }}
          style={[inputStyles(type, value), inputErrorStyles(error), style]}
          type={type}
          {...inputOptions}
          value={value}
        />
      </div>
    );
  }
}

export default radium(InputField);

