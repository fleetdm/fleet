import React, { Component, PropTypes } from 'react';
import { isEqual, noop } from 'lodash';

const defaultValidate = () => { return { valid: true, errors: {} }; };

export default (WrappedComponent, { fields, validate = defaultValidate }) => {
  class Form extends Component {
    static propTypes = {
      errors: PropTypes.object, // eslint-disable-line react/forbid-prop-types
      formData: PropTypes.object, // eslint-disable-line react/forbid-prop-types
      handleSubmit: PropTypes.func,
      onChangeFunc: PropTypes.func,
    };

    static defaultProps = {
      errors: {},
      formData: {},
    };

    constructor (props) {
      super(props);

      const { errors, formData } = props;

      this.state = { errors, formData };
    }

    componentWillReceiveProps (nextProps) {
      const { formData: formDataProp } = nextProps;
      const { formData: oldFormDataProp } = this.props;

      if (!isEqual(formDataProp, oldFormDataProp)) {
        const { formData } = this.state;

        this.setState({
          formData: {
            ...formData,
            ...formDataProp,
          },
        });
      }

      return false;
    }

    onFieldChange = (fieldName) => {
      return (value) => {
        const { errors, formData } = this.state;
        const { onChangeFunc = noop } = this.props;

        onChangeFunc(fieldName, value);

        this.setState({
          errors: { ...errors, [fieldName]: null },
          formData: { ...formData, [fieldName]: value },
        });

        return false;
      };
    }

    onSubmit = (evt) => {
      evt.preventDefault();

      const { handleSubmit } = this.props;
      const { errors, formData } = this.state;
      const { valid, errors: clientErrors } = validate(formData);

      if (valid) {
        return handleSubmit(formData);
      }

      this.setState({
        errors: { ...errors, ...clientErrors },
      });

      return false;
    }

    getError = (fieldName) => {
      const { errors } = this.state;
      const { errors: serverErrors } = this.props;

      return errors[fieldName] || serverErrors[fieldName];
    }

    getFields = () => {
      const { getError, getValue, onFieldChange } = this;
      const fieldProps = {};

      fields.forEach((field) => {
        fieldProps[field] = {
          error: getError(field),
          name: field,
          onChange: onFieldChange(field),
          value: getValue(field),
        };
      });

      return fieldProps;
    }

    getValue = (fieldName) => {
      return this.state.formData[fieldName];
    }

    render () {
      const { getFields, onSubmit, props } = this;

      return <WrappedComponent {...props} fields={getFields()} handleSubmit={onSubmit} />;
    }
  }

  return Form;
};
