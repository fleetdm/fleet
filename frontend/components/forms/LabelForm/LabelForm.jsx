import React, { Component } from 'react';
import PropTypes from 'prop-types';

import Button from 'components/buttons/Button';
import Dropdown from 'components/forms/fields/Dropdown';
import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import helpers from 'components/forms/queries/QueryForm/helpers';
import InputField from 'components/forms/fields/InputField';
import KolideAce from 'components/KolideAce';
import validate from 'components/forms/LabelForm/validate';

const baseClass = 'label-form';

class LabelForm extends Component {
  static propTypes = {
    baseError: PropTypes.string,
    fields: PropTypes.shape({
      description: formFieldInterface.isRequired,
      name: formFieldInterface.isRequired,
      platform: formFieldInterface.isRequired,
      query: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
    isEdit: PropTypes.bool,
    onCancel: PropTypes.func.isRequired,
    onOsqueryTableSelect: PropTypes.func,
  };

  static defaultProps = {
    isEdit: false,
  };

  onLoad = (editor) => {
    editor.setOptions({
      enableLinking: true,
    });

    editor.on('linkClick', (data) => {
      const { type, value } = data.token;
      const { onOsqueryTableSelect } = this.props;

      if (type === 'osquery-token') {
        return onOsqueryTableSelect(value);
      }

      return false;
    });
  }

  render () {
    const { baseError, fields, handleSubmit, isEdit, onCancel } = this.props;
    const { onLoad } = this;
    const headerText = isEdit ? 'Edit Label' : 'New Label Query';
    const saveBtnText = isEdit ? 'Update Label' : 'Save Label';

    return (
      <form className={`${baseClass}__wrapper`} onSubmit={handleSubmit}>
        <h1>{headerText}</h1>
        <KolideAce
          {...fields.query}
          onLoad={onLoad}
          readOnly={isEdit}
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
        />
        {baseError && <div className="form__base-error">{baseError}</div>}
        <InputField
          {...fields.name}
          inputClassName={`${baseClass}__label-title`}
          label="Label title"
        />
        <InputField
          {...fields.description}
          inputClassName={`${baseClass}__label-description`}
          label="Description"
          type="textarea"
        />
        <div className="form-field form-field--dropdown">
          <label className="form-field__label" htmlFor="platform">Platform</label>
          <Dropdown
            {...fields.platform}
            options={helpers.platformOptions}
          />
        </div>
        <div className={`${baseClass}__button-wrap`}>
          <Button
            className={`${baseClass}__cancel-btn`}
            onClick={onCancel}
            variant="inverse"
          >
            Cancel
          </Button>
          <Button
            className={`${baseClass}__save-btn`}
            type="submit"
            variant="brand"
          >
            {saveBtnText}
          </Button>
        </div>
      </form>
    );
  }
}

export default Form(LabelForm, {
  fields: ['description', 'name', 'platform', 'query'],
  validate,
});

