import React, { Component, PropTypes } from 'react';

import Avatar from '../../../Avatar';
import Button from '../../../buttons/Button';
import InputField from '../../fields/InputField';
import userInterface from '../../../../interfaces/user';

const baseClass = 'edit-user-form';

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
    const { user } = this.props;
    const {
      email,
      name,
      position,
      username,
    } = user;
    const { onFormSubmit, onInputChange } = this;

    return (
      <form className={baseClass} onSubmit={onFormSubmit}>
        <InputField
          defaultValue={name}
          label="name"
          labelClassName={`${baseClass}__label`}
          name="name"
          onChange={onInputChange('name')}
          inputWrapperClass={`${baseClass}__input-wrap ${baseClass}__input-wrap--first`}
          inputClassName={`${baseClass}__input`}
        />
        <div className={`${baseClass}__avatar-wrap`}>
          <Avatar user={user} className="user-block__avatar" />
        </div>
        <InputField
          defaultValue={username}
          label="username"
          labelClassName={`${baseClass}__label`}
          name="username"
          onChange={onInputChange('username')}
          inputWrapperClass={`${baseClass}__input-wrap`}
          inputClassName={`${baseClass}__input ${baseClass}__input--username`}
        />
        <InputField
          defaultValue={position}
          label="position"
          labelClassName={`${baseClass}__label`}
          name="position"
          onChange={onInputChange('position')}
          inputWrapperClass={`${baseClass}__input-wrap`}
          inputClassName={`${baseClass}__input`}
        />
        <InputField
          defaultValue={email}
          inputWrapperClass={`${baseClass}__input-wrap`}
          label="email"
          labelClassName={`${baseClass}__label`}
          name="email"
          onChange={onInputChange('email')}
          inputClassName={`${baseClass}__input ${baseClass}__input--email`}
        />
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__form-btn`}
            text="Submit"
            type="submit"
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
