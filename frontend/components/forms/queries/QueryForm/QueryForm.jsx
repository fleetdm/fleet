import React, { Component, PropTypes } from 'react';
import { size } from 'lodash';

import Button from 'components/buttons/Button';
import DropdownButton from 'components/buttons/DropdownButton';
import Form from 'components/forms/Form';
import formFieldInterface from 'interfaces/form_field';
import helpers from 'components/forms/queries/QueryForm/helpers';
import InputField from 'components/forms/fields/InputField';
import KolideAce from 'components/KolideAce';
import queryInterface from 'interfaces/query';
import SelectTargetsDropdown from 'components/forms/fields/SelectTargetsDropdown';
import targetInterface from 'interfaces/target';
import validateQuery from 'components/forms/validators/validate_query';
import Timer from 'components/loaders/Timer';

const baseClass = 'query-form';

const validate = (formData) => {
  const errors = {};
  const {
    error: queryError,
    valid: queryValid,
  } = validateQuery(formData.query);

  if (!queryValid) {
    errors.query = queryError;
  }

  if (!formData.name) {
    errors.name = 'Title must be present';
  }

  const valid = !size(errors);

  return { valid, errors };
};

class QueryForm extends Component {
  static propTypes = {
    baseError: PropTypes.string,
    fields: PropTypes.shape({
      description: formFieldInterface.isRequired,
      name: formFieldInterface.isRequired,
      query: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func,
    formData: queryInterface,
    onFetchTargets: PropTypes.func,
    onOsqueryTableSelect: PropTypes.func,
    onRunQuery: PropTypes.func,
    onStopQuery: PropTypes.func,
    onTargetSelect: PropTypes.func,
    onUpdate: PropTypes.func,
    queryIsRunning: PropTypes.bool,
    selectedTargets: PropTypes.arrayOf(targetInterface),
    targetsCount: PropTypes.number,
    targetsError: PropTypes.string,
  };

  static defaultProps = {
    targetsCount: 0,
  };

  constructor (props) {
    super(props);

    this.state = { errors: {} };
  }

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

  onRunQuery = (queryText) => {
    return (evt) => {
      evt.preventDefault();

      const { onRunQuery: handleRunQuery } = this.props;

      return handleRunQuery(queryText);
    };
  }

  onUpdate = (evt) => {
    evt.preventDefault();

    const { fields } = this.props;
    const { onUpdate: handleUpdate } = this.props;
    const formData = {
      description: fields.description.value,
      name: fields.name.value,
      query: fields.query.value,
    };

    const { valid, errors } = validate(formData);

    if (valid) {
      handleUpdate(formData);

      return false;
    }

    this.setState({
      errors: {
        ...this.state.errors,
        ...errors,
      },
    });

    return false;
  }

  renderButtons = () => {
    const { canSaveAsNew, canSaveChanges } = helpers;
    const {
      fields,
      formData,
      handleSubmit,
      onStopQuery,
      queryIsRunning,
    } = this.props;
    const { onRunQuery, onUpdate } = this;

    const dropdownBtnOptions = [{
      disabled: !canSaveChanges(fields, formData),
      label: 'Save Changes',
      onClick: onUpdate,
    }, {
      disabled: !canSaveAsNew(fields, formData),
      label: 'Save As New...',
      onClick: handleSubmit,
    }];

    let runQueryButton;

    if (queryIsRunning) {
      runQueryButton = (
        <Button
          className={`${baseClass}__stop-query-btn`}
          onClick={onStopQuery}
          variant="alert"
        >
          Stop Query
        </Button>
      );
    } else {
      runQueryButton = (
        <Button
          className={`${baseClass}__run-query-btn`}
          onClick={onRunQuery(fields.query.value)}
          variant="brand"
        >
          Run Query
        </Button>
      );
    }

    return (
      <div className={`${baseClass}__button-wrap`}>
        {queryIsRunning && <Timer running={queryIsRunning} />}

        <DropdownButton
          className={`${baseClass}__save`}
          options={dropdownBtnOptions}
          variant="success"
        >
          Save
        </DropdownButton>

        {runQueryButton}
      </div>
    );
  }

  renderTargetsInput = () => {
    const {
      onFetchTargets,
      onTargetSelect,
      selectedTargets,
      targetsCount,
      targetsError,
    } = this.props;


    return (
      <div>
        <SelectTargetsDropdown
          error={targetsError}
          onFetchTargets={onFetchTargets}
          onSelect={onTargetSelect}
          selectedTargets={selectedTargets}
          targetsCount={targetsCount}
          label="Select Targets"
        />
      </div>
    );
  }

  render () {
    const { errors } = this.state;
    const { baseError, fields, handleSubmit, queryIsRunning } = this.props;
    const { onLoad, renderButtons, renderTargetsInput } = this;

    return (
      <form className={`${baseClass}__wrapper`} onSubmit={handleSubmit}>
        <h1>New Query</h1>
        <KolideAce
          {...fields.query}
          error={fields.query.error || errors.query}
          onLoad={onLoad}
          readOnly={queryIsRunning}
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
        />
        {baseError && <div className="form__base-error">{baseError}</div>}
        {renderTargetsInput()}
        <InputField
          {...fields.name}
          error={fields.name.error || errors.name}
          inputClassName={`${baseClass}__query-title`}
          label="Query Title"
        />
        <InputField
          {...fields.description}
          inputClassName={`${baseClass}__query-description`}
          label="Description"
          type="textarea"
        />
        {renderButtons()}
      </form>
    );
  }
}

export default Form(QueryForm, {
  fields: ['description', 'name', 'platform', 'query'],
  validate,
});
