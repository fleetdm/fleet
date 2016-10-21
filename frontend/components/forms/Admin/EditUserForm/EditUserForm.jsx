import React, { Component, PropTypes } from 'react';
import radium from 'radium';

import Avatar from '../../../Avatar';
import { avatarStyles } from '../../../../pages/Admin/UserManagementPage/UserBlock/styles';
import Button from '../../../buttons/Button';
import componentStyles from './styles';
import InputField from '../../fields/InputField';
import Styleguide from '../../../../styles';
import userInterface from '../../../../interfaces/user';

const { color } = Styleguide;

class EditUserForm extends Component {
  static propTypes = {
    onCancel: PropTypes.func,
    onSubmit: PropTypes.func,
    user: userInterface,
  };

  constructor (props) {
    super(props);

    this.state = {
      formData: {},
    };
  }

  onInputChange = (fieldName) => {
    return (evt) => {
      const { formData } = this.state;

      this.setState({
        formData: {
          ...formData,
          [fieldName]: evt.target.value,
        },
      });

      return false;
    };
  }

  onFormSubmit = (evt) => {
    evt.preventDefault();
    const { formData } = this.state;
    const { onSubmit } = this.props;

    return onSubmit(formData);
  }

  render () {
    const {
      avatarWrapperStyles,
      buttonWrapperStyles,
      formButtonStyles,
      formWrapperStyles,
      inputStyles,
      inputWrapperStyles,
      labelStyles,
    } = componentStyles;
    const { user } = this.props;
    const {
      email,
      name,
      position,
      username,
    } = user;
    const { onFormSubmit, onInputChange } = this;

    return (
      <form style={formWrapperStyles} onSubmit={onFormSubmit}>
        <InputField
          defaultValue={name}
          label="name"
          labelStyles={labelStyles}
          name="name"
          onChange={onInputChange('name')}
          inputWrapperStyles={inputWrapperStyles}
          style={inputStyles}
        />
        <div style={avatarWrapperStyles}>
          <Avatar user={user} style={avatarStyles} />
        </div>
        <InputField
          defaultValue={username}
          label="username"
          labelStyles={labelStyles}
          name="username"
          onChange={onInputChange('username')}
          inputWrapperStyles={{ marginTop: 0 }}
          style={[inputStyles, { color: color.brand }]}
        />
        <InputField
          defaultValue={position}
          label="position"
          labelStyles={labelStyles}
          name="position"
          onChange={onInputChange('position')}
          inputWrapperStyles={{ marginTop: 0 }}
          style={inputStyles}
        />
        <InputField
          defaultValue={email}
          inputWrapperStyles={{ marginTop: 0 }}
          label="email"
          labelStyles={labelStyles}
          name="email"
          onChange={onInputChange('email')}
          style={[inputStyles, { color: color.link }]}
        />
        <div style={buttonWrapperStyles}>
          <Button
            style={formButtonStyles}
            text="Submit"
            type="submit"
          />
          <Button
            onClick={this.props.onCancel}
            style={formButtonStyles}
            text="Cancel"
            variant="inverse"
          />
        </div>
      </form>
    );
  }
}

export default radium(EditUserForm);
