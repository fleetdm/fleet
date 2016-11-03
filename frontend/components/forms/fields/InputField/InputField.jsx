import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';

const baseClass = 'input-field';

class InputField extends Component {
  static propTypes = {
    autofocus: PropTypes.bool,
    defaultValue: PropTypes.string,
    error: PropTypes.string,
    inputClassName: PropTypes.string, // eslint-disable-line react/forbid-prop-types
    inputWrapperClass: PropTypes.string,
    inputOptions: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    label: PropTypes.string,
    labelClassName: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func,
    placeholder: PropTypes.string,
    type: PropTypes.string,
  };

  static defaultProps = {
    autofocus: false,
    defaultValue: '',
    inputWrapperClass: '',
    inputOptions: {},
    label: null,
    labelClassName: '',
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
    const { error, label, labelClassName, name } = this.props;
    const labelWrapperClasses = classnames(
      `${baseClass}__label`,
      labelClassName,
      { [`${baseClass}__label--error`]: error }
    );

    if (!label) {
      return false;
    }

    return (
      <label
        className={labelWrapperClasses}
        htmlFor={name}
      >
        {error || label}
      </label>
    );
  }

  render () {
    const { error, inputClassName, inputOptions, inputWrapperClass, name, placeholder, type } = this.props;
    const { value } = this.state;
    const { onInputChange, renderLabel } = this;
    const inputClasses = classnames(
      baseClass,
      inputClassName,
      { [`${baseClass}--password`]: type === 'password' && value },
      { [`${baseClass}--error`]: error }
    );
    const inputWrapperClasses = classnames(`${baseClass}__wrapper`, inputWrapperClass);

    if (type === 'textarea') {
      return (
        <div className={inputWrapperClasses}>
          {renderLabel()}
          <textarea
            name={name}
            onChange={onInputChange}
            className={inputClasses}
            placeholder={placeholder}
            ref={(r) => { this.input = r; }}
            type={type}
            {...inputOptions}
            value={value}
          />
        </div>
      );
    }

    return (
      <div className={inputWrapperClasses}>
        {renderLabel()}
        <input
          name={name}
          onChange={onInputChange}
          className={inputClasses}
          placeholder={placeholder}
          ref={(r) => { this.input = r; }}
          type={type}
          {...inputOptions}
          value={value}
        />
      </div>
    );
  }
}

export default InputField;

