import React, { Component, PropTypes } from 'react';
import radium from 'radium';
import componentStyles from './styles';
import Icon from '../../icons/Icon';
import InputFieldWithIcon from '../fields/InputFieldWithIcon';

class LoginForm extends Component {
  static propTypes = {
    onSubmit: PropTypes.func,
  };

  constructor (props) {
    super(props);

    this.state = {
      formData: {
        email: null,
        password: null,
      },
    };
  }

  onInputChange = (formField) => {
    return ({ target }) => {
      const { formData } = this.state;
      const { value } = target;

      this.setState({
        formData: {
          ...formData,
          [formField]: value,
        },
      });
    };
  }

  onFormSubmit = () => {
    const { formData } = this.state;
    const { onSubmit } = this.props;

    return onSubmit(formData);
  }

  render () {
    const { containerStyles, submitButtonStyles, userIconStyles } = componentStyles;
    const { onInputChange, onFormSubmit } = this;
    const { formData } = this.state;
    const canSubmit = formData.email && formData.password;

    return (
      <div>
        <div style={containerStyles}>
          <Icon name="user" variant="circle" style={userIconStyles} />
          <InputFieldWithIcon
            iconName="user"
            name="email"
            onChange={onInputChange('email')}
            placeholder="Username or Email"
          />
          <InputFieldWithIcon
            iconName="lock"
            name="password"
            onChange={onInputChange('password')}
            placeholder="Password"
            type="password"
          />
        </div>
        <button
          disabled={!canSubmit}
          onClick={onFormSubmit}
          style={submitButtonStyles(canSubmit)}
        >
          Login
        </button>
      </div>
    );
  }
}

export default radium(LoginForm);
