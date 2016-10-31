import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';

import Button from '../../../buttons/Button';
import InputField from '../../fields/InputField';
import validatePresence from '../../validators/validate_presence';

const baseClass = 'save-query-form';

class SaveQueryForm extends Component {
  static propTypes = {
    onCancel: PropTypes.func,
    onSubmit: PropTypes.func,
  };

  constructor (props) {
    super(props);

    this.state = {
      errors: {
        name: null,
        description: null,
      },
      formData: {
        name: null,
        description: null,
      },
    };
  }

  onFieldChange = (fieldName) => {
    return ({ target }) => {
      const { errors, formData } = this.state;
      this.setState({
        errors: {
          ...errors,
          [fieldName]: null,
        },
        formData: {
          ...formData,
          [fieldName]: target.value,
        },
      });
    };
  }

  onFormSubmit = (evt) => {
    evt.preventDefault();

    const { formData } = this.state;
    const { onSubmit } = this.props;
    const { validate } = this;

    if (validate()) {
      return onSubmit({ ...formData });
    }

    return false;
  }

  validate = () => {
    const { errors, formData: { name } } = this.state;

    if (!validatePresence(name)) {
      this.setState({
        errors: {
          ...errors,
          name: 'Query Name field must be completed',
        },
      });

      return false;
    }

    return true;
  }

  render () {
    const { errors } = this.state;
    const { onCancel } = this.props;
    const { onFieldChange, onFormSubmit } = this;
    const nameInputClassName = classnames(`${baseClass}__input`, `${baseClass}__input--name`);
    const descriptionInputClassName = classnames(`${baseClass}__input`, `${baseClass}__input--description`);

    return (
      <form onSubmit={onFormSubmit}>
        <InputField
          error={errors.name}
          inputClassName={nameInputClassName}
          label="Query Name"
          labelClassName={`${baseClass}__label`}
          name="name"
          onChange={onFieldChange('name')}
          placeholder="e.g. Interesting Query Name"
        />
        <InputField
          error={errors.description}
          inputClassName={descriptionInputClassName}
          label="Query Description"
          labelClassName={`${baseClass}__label`}
          name="description"
          onChange={onFieldChange('description')}
          placeholder="e.g. This query does x, y, & z because n"
          type="textarea"
        />
        <div className={`${baseClass}__btn--wrapper`}>
          <Button
            className={`${baseClass}__btn--cancel`}
            onClick={onCancel}
            text="Cancel"
            variant="inverse"
          />
          <Button
            className={`${baseClass}__btn--submit`}
            text="Save Query"
            type="submit"
          />
        </div>
      </form>
    );
  }
}

export default SaveQueryForm;
