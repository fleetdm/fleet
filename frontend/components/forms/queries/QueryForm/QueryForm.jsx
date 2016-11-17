import React, { Component, PropTypes } from 'react';
import { isEqual } from 'lodash';

import Button from 'components/buttons/Button';
import Dropdown from 'components/forms/fields/Dropdown';
import helpers from 'components/forms/queries/QueryForm/helpers';
import InputField from 'components/forms/fields/InputField';
import queryInterface from 'interfaces/query';
import validatePresence from 'components/forms/validators/validate_presence';

const baseClass = 'query-form';

class QueryForm extends Component {
  static propTypes = {
    onCancel: PropTypes.func,
    onRunQuery: PropTypes.func,
    onSave: PropTypes.func,
    onUpdate: PropTypes.func,
    query: queryInterface,
    queryText: PropTypes.string.isRequired,
    queryType: PropTypes.string,
  };

  static defaultProps = {
    query: {},
  };

  constructor (props) {
    super(props);

    const {
      query: { description, name },
      queryText,
      queryType,
    } = this.props;
    const errors = { description: null, name: null };
    const formData = { description, name, query: queryText };

    if (queryType === 'label') {
      const { allPlatforms: platform } = helpers;

      this.state = {
        errors: { ...errors, platform: null },
        formData: { ...formData, platform },
      };
    } else {
      this.state = { errors, formData };
    }
  }

  componentDidMount = () => {
    const { query, queryText } = this.props;
    const { description, name } = query;
    const { formData } = this.state;

    this.setState({
      formData: {
        ...formData,
        description,
        name,
        query: queryText,
      },
    });
  }

  componentWillReceiveProps = (nextProps) => {
    const { query, queryText } = nextProps;
    const { query: staleQuery, queryText: staleQueryText } = this.props;

    if (!isEqual(query, staleQuery) || !isEqual(queryText, staleQueryText)) {
      const { formData } = this.state;

      this.setState({
        formData: {
          ...formData,
          description: query.description || formData.description,
          name: query.name || formData.name,
          query: queryText,
        },
      });
    }

    return false;
  }

  onCancel = (evt) => {
    evt.preventDefault();

    const { onCancel: handleCancel } = this.props;

    return handleCancel();
  }

  onFieldChange = (name) => {
    return (value) => {
      const { errors, formData } = this.state;

      this.setState({
        errors: {
          ...errors,
          [name]: null,
        },
        formData: {
          ...formData,
          [name]: value,
        },
      });

      return false;
    };
  }

  onSave = (evt) => {
    evt.preventDefault();

    const { formData } = this.state;
    const { valid } = this;
    const { onSave: handleSave } = this.props;

    if (valid()) {
      handleSave(formData);
    }

    return false;
  }

  onUpdate = (evt) => {
    evt.preventDefault();

    const { formData } = this.state;
    const { valid } = this;
    const { onUpdate: handleUpdate } = this.props;

    if (valid()) {
      handleUpdate(formData);
    }

    return false;
  }

  valid = () => {
    const { errors, formData: { name } } = this.state;
    const { queryType } = this.props;

    const namePresent = validatePresence(name);

    if (!namePresent) {
      this.setState({
        errors: {
          ...errors,
          name: `${queryType === 'label' ? 'Label title' : 'Query title'} must be present`,
        },
      });

      return false;
    }

    // TODO: validate queryText

    return true;
  }

  renderButtons = () => {
    const { canSaveAsNew, canSaveChanges } = helpers;
    const { formData } = this.state;
    const { onRunQuery, query, queryType } = this.props;
    const { onCancel, onSave, onUpdate } = this;

    if (queryType === 'label') {
      return (
        <div className={`${baseClass}__button-wrap`}>
          <Button
            className={`${baseClass}__save-changes-btn`}
            onClick={onCancel}
            text="Cancel"
            variant="inverse"
          />
          <Button
            className={`${baseClass}__save-as-new-btn`}
            disabled={!canSaveAsNew(formData, query)}
            onClick={onSave}
            text="Save Label"
            variant="brand"
          />
        </div>
      );
    }

    return (
      <div className={`${baseClass}__button-wrap`}>
        <Button
          className={`${baseClass}__save-changes-btn`}
          disabled={!canSaveChanges(formData, query)}
          onClick={onUpdate}
          text="Save Changes"
          variant="inverse"
        />
        <Button
          className={`${baseClass}__save-as-new-btn`}
          disabled={!canSaveAsNew(formData, query)}
          onClick={onSave}
          text="Save As New..."
          variant="success"
        />
        <Button
          className={`${baseClass}__run-query-btn`}
          onClick={onRunQuery}
          text="Run Query"
          variant="brand"
        />
      </div>
    );
  }

  renderPlatformDropdown = () => {
    const { queryType } = this.props;

    if (queryType !== 'label') {
      return false;
    }

    const { formData: { platform } } = this.state;
    const { onFieldChange } = this;
    const { platformOptions } = helpers;

    return (
      <Dropdown
        options={platformOptions}
        onSelect={onFieldChange('platform')}
        value={platform.value}
      />
    );
  }

  render () {
    const {
      errors,
      formData: {
        description,
        name,
      },
    } = this.state;
    const { onFieldChange, renderPlatformDropdown, renderButtons } = this;
    const { queryType } = this.props;

    return (
      <form className={baseClass}>
        <InputField
          error={errors.name}
          label={queryType === 'label' ? 'Label title' : 'Query Title'}
          name="name"
          onChange={onFieldChange('name')}
          value={name}
          inputClassName={`${baseClass}__query-title`}
        />
        <InputField
          error={errors.description}
          label="Description"
          name="description"
          onChange={onFieldChange('description')}
          value={description}
          type="textarea"
          inputClassName={`${baseClass}__query-description`}
        />
        {renderPlatformDropdown()}
        {renderButtons()}
      </form>
    );
  }
}

export default QueryForm;
