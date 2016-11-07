import React, { Component, PropTypes } from 'react';
import { isEqual } from 'lodash';

import Button from 'components/buttons/Button';
import helpers from 'components/forms/queries/QueryForm/helpers';
import InputField from 'components/forms/fields/InputField';
import queryInterface from 'interfaces/query';
import validatePresence from 'components/forms/validators/validate_presence';

const baseClass = 'query-form';

class QueryForm extends Component {
  static propTypes = {
    onRunQuery: PropTypes.func,
    onSaveAsNew: PropTypes.func,
    onSaveChanges: PropTypes.func,
    query: queryInterface,
    queryText: PropTypes.string.isRequired,
  };

  static defaultProps = {
    query: {},
  };

  constructor (props) {
    super(props);

    const { query: { description, name }, queryText } = this.props;

    this.state = {
      errors: {
        description: null,
        name: null,
      },
      formData: {
        description,
        name,
        queryText,
      },
    };
  }

  componentDidMount = () => {
    const { query, queryText } = this.props;
    const { description, name } = query;

    this.setState({
      formData: {
        description,
        name,
        queryText,
      },
    });
  }

  componentWillReceiveProps = (nextProps) => {
    const { query, queryText } = nextProps;
    const { query: staleQuery, queryText: staleQueryText } = this.props;

    if (!isEqual(query, staleQuery) || !isEqual(queryText, staleQueryText)) {
      this.setState({
        formData: {
          description: query.description,
          name: query.name,
          queryText,
        },
      });
    }

    return false;
  }

  onFieldChange = (name) => {
    return (value) => {
      const { formData } = this.state;

      this.setState({
        formData: {
          ...formData,
          [name]: value,
        },
      });

      return false;
    };
  }

  onSaveAsNew = (evt) => {
    evt.preventDefault();

    const { formData } = this.state;
    const { valid } = this;
    const { onSaveAsNew: handleSaveAsNew } = this.props;

    if (valid()) {
      handleSaveAsNew(formData);
    }

    return false;
  }

  onSaveChanges = (evt) => {
    evt.preventDefault();

    const { formData } = this.state;
    const { valid } = this;
    const { onSaveChanges: handleSaveChanges } = this.props;

    if (valid()) {
      handleSaveChanges(formData);
    }

    return false;
  }

  valid = () => {
    const { errors, formData: { name } } = this.state;

    const namePresent = validatePresence(name);

    if (!namePresent) {
      this.setState({
        errors: {
          ...errors,
          name: 'Query name must be present',
        },
      });

      return false;
    }

    // TODO: validate queryText

    return true;
  }

  render () {
    const {
      errors,
      formData,
      formData: {
        description,
        name,
      },
    } = this.state;
    const { onFieldChange, onSaveAsNew, onSaveChanges } = this;
    const { onRunQuery, query } = this.props;
    const { canSaveAsNew, canSaveChanges } = helpers;

    return (
      <form>
        <InputField
          error={errors.name}
          label="Query Title:"
          name="name"
          onChange={onFieldChange('name')}
          value={name}
        />
        <InputField
          error={errors.description}
          label="Query Description:"
          name="description"
          onChange={onFieldChange('description')}
          value={description}
        />
        <Button
          className={`${baseClass}__save-changes-btn`}
          disabled={!canSaveChanges(formData, query)}
          onClick={onSaveChanges}
          text="Save Changes"
          variant="inverse"
        />
        <Button
          className={`${baseClass}__save-as-new-btn`}
          disabled={!canSaveAsNew(formData, query)}
          onClick={onSaveAsNew}
          text="Save As New..."
        />
        <Button
          className={`${baseClass}__run-query-btn`}
          onClick={onRunQuery}
          text="Run Query"
        />
      </form>
    );
  }
}

export default QueryForm;
