import React, { Component, PropTypes } from 'react';

import Button from '../../../buttons/Button';
import InputField from '../../fields/InputField';
import userInterface from '../../../../interfaces/user';

const baseClass = 'edit-user-form';

class EditUserForm extends Component {
  static propTypes = {
    onCancel: PropTypes.func,
    onSubmit: PropTypes.func,
    user: userInterface.isRequired,
  };

  constructor (props) {
    super(props);

    const { user: { email, name, position, username } } = props;

    this.state = {
      formData: {
        email,
        name,
        position,
        username,
      },
    };
  }

  onInputChange = (fieldName) => {
    return (value) => {
      const { formData } = this.state;

      this.setState({
        formData: {
          ...formData,
          [fieldName]: value,
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
      email,
      name,
      position,
      username,
    } = this.state.formData;
    const { onFormSubmit, onInputChange } = this;

    return (
      <form className={baseClass} onSubmit={onFormSubmit}>
        <InputField
          defaultValue={name}
          label="Name"
          labelClassName={`${baseClass}__label`}
          name="name"
          onChange={onInputChange('name')}
          inputWrapperClass={`${baseClass}__input-wrap ${baseClass}__input-wrap--first`}
          inputClassName={`${baseClass}__input`}
          value={name}
        />
        <InputField
          defaultValue={username}
          label="Username"
          labelClassName={`${baseClass}__label`}
          name="username"
          onChange={onInputChange('username')}
          inputWrapperClass={`${baseClass}__input-wrap`}
          inputClassName={`${baseClass}__input ${baseClass}__input--username`}
          value={username}
        />
        <InputField
          defaultValue={position}
          label="Position"
          labelClassName={`${baseClass}__label`}
          name="position"
          onChange={onInputChange('position')}
          inputWrapperClass={`${baseClass}__input-wrap`}
          inputClassName={`${baseClass}__input`}
          value={position}
        />
        <InputField
          defaultValue={email}
          inputWrapperClass={`${baseClass}__input-wrap`}
          label="Email"
          labelClassName={`${baseClass}__label`}
          name="email"
          onChange={onInputChange('email')}
          inputClassName={`${baseClass}__input ${baseClass}__input--email`}
          value={email}
        />
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__form-btn ${baseClass}__form-btn--submit`}
            text="Submit"
            type="submit"
            variant="brand"
          />
          <Button
            className={`${baseClass}__form-btn`}
            onClick={this.props.onCancel}
            text="Cancel"
            variant="inverse"
          />
        </div>
      </form>
    );
  }
}

export default EditUserForm;
