import React, { Component, PropTypes } from 'react';
import Form from 'components/forms/Form';
import KolideAce from 'components/KolideAce';
import Dropdown from 'components/forms/fields/Dropdown';
import InputField from 'components/forms/fields/InputField';
import Button from 'components/buttons/Button';
import formFieldInterface from 'interfaces/form_field';
import validateQuery from 'components/forms/validators/validate_query';
import { size } from 'lodash';

const baseClass = 'decorator-form';

const validate = (formData) => {
  const errors = {};
  const {
    error: queryError,
    valid: queryValid,
  } = validateQuery(formData.query);
  if (!queryValid) {
    errors.query = queryError;
  }
  if (formData.name == null || formData.name === '') {
    errors.name = 'Name can not be empty';
  }
  // interval value must be evenly divisible by 60
  if (formData.type === 'interval') {
    if ((formData.interval % 60) !== 0) {
      errors.interval = 'Interval must be evenly divisible by 60';
    } else if (formData.interval <= 0) {
      errors.interval = 'Interval must be greater than zero';
    }
  }
  const valid = !size(errors);
  return { valid, errors };
};

class DecoratorForm extends Component {
  static propTypes = {
    fields: PropTypes.shape({
      id: formFieldInterface,
      query: formFieldInterface,
      interval: formFieldInterface,
      type: formFieldInterface,
      name: formFieldInterface,
      built_in: formFieldInterface,
    }),
    handleCancel: PropTypes.func,
    handleSubmit: PropTypes.func,
    newDecorator: PropTypes.bool,
  };

  constructor (props) {
    super(props);
    this.state = { errors: {} };
  }

  render() {
    const { handleSubmit, handleCancel, fields, newDecorator } = this.props;
    const { type } = fields;
    const { errors } = this.state;
    const types = [
      { label: 'Load', value: 'load' },
      { label: 'Always', value: 'always' },
      { label: 'Interval', value: 'interval' },
    ];
    const formTitle = newDecorator ? 'New Osquery Decorator' : 'Edit Osquery Decorator';


    return (
      <form className={`${baseClass}__wrapper`} onSubmit={handleSubmit} >
        <h1>{formTitle}</h1>
        <InputField
          {...fields.name}
          label="Decorator Name"
          inputClassName={`${baseClass}__name`}
        />
        <KolideAce
          {...fields.query}
          error={fields.query.error || errors.query}
          label="SQL"
        />
        <div className={`${baseClass}__inputs`}>
          <Dropdown
            {...fields.type}
            options={types}
            label="Decorator Type"
            wrapperClassName={`${baseClass}__dropdown`}
          />
          <InputField
            {...fields.interval}
            label="Interval Duration"
            disabled={type.value !== 'interval'}
            type="number"
          />
        </div>
        <div className={`${baseClass}__button-wrap`}>
          <Button
            className={`${baseClass}__form-btn ${baseClass}__form-btn--submit`}
            type="submit"
            onClick={handleSubmit}
          >
            Submit
          </Button>
          <Button
            className={`${baseClass}__form-btn`}
            type="inverse"
            onClick={handleCancel}
          >
            Cancel
          </Button>
        </div>
      </form>
    );
  }
}

export default Form(DecoratorForm, {
  fields: ['id', 'name', 'type', 'query', 'interval', 'built_in'],
  validate,
});
