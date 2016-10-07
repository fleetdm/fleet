import React, { Component, PropTypes } from 'react';
import radium from 'radium';
import Avatar from '../../../Avatar';
import Button from '../../../buttons/Button';
import componentStyles from '../../../../pages/Admin/UserManagementPage/UserBlock/styles';
import InputField from '../../fields/InputField';
import Styleguide from '../../../../styles';

const { color, font, padding } = Styleguide;

class EditUserForm extends Component {
  static propTypes = {
    onCancel: PropTypes.func,
    onSubmit: PropTypes.func,
    user: PropTypes.object,
  };

  static inputStyles = {
    borderLeft: 'none',
    borderRight: 'none',
    borderTop: 'none',
    borderBottomWidth: '1px',
    fontSize: font.small,
    borderBottomStyle: 'solid',
    borderBottomColor: color.brand,
    color: color.textMedium,
    width: '100%',
  };

  static labelStyles = {
    color: color.textLight,
    textTransform: 'uppercase',
    fontSize: font.mini,
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
      avatarStyles,
      formButtonStyles,
      userWrapperStyles,
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
      <form style={[userWrapperStyles, { boxSizing: 'border-box', padding: '10px' }]} onSubmit={onFormSubmit}>
        <InputField
          defaultValue={name}
          label="name"
          labelStyles={EditUserForm.labelStyles}
          name="name"
          onChange={onInputChange('name')}
          inputWrapperStyles={{ marginTop: 0, marginBottom: padding.half }}
          style={EditUserForm.inputStyles}
        />
        <Avatar user={user} style={avatarStyles} />
        <InputField
          defaultValue={username}
          label="username"
          labelStyles={EditUserForm.labelStyles}
          name="username"
          onChange={onInputChange('username')}
          inputWrapperStyles={{ marginTop: 0 }}
          style={[EditUserForm.inputStyles, { color: color.brand }]}
        />
        <InputField
          defaultValue={position}
          label="position"
          labelStyles={EditUserForm.labelStyles}
          name="position"
          onChange={onInputChange('position')}
          inputWrapperStyles={{ marginTop: 0 }}
          style={EditUserForm.inputStyles}
        />
        <InputField
          defaultValue={email}
          inputWrapperStyles={{ marginTop: 0 }}
          label="email"
          labelStyles={EditUserForm.labelStyles}
          name="email"
          onChange={onInputChange('email')}
          style={[EditUserForm.inputStyles, { color: color.link }]}
        />
        <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '10px' }}>
          <Button
            onClick={this.props.onCancel}
            style={formButtonStyles}
            text="Cancel"
            variant="inverse"
          />
          <Button
            style={formButtonStyles}
            text="Submit"
            type="submit"
          />
        </div>
      </form>
    );
  }
}

export default radium(EditUserForm);
