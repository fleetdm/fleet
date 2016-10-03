import React, { Component, PropTypes } from 'react';
import radium from 'radium';
import componentStyles from './styles';
import Button from '../../../buttons/Button';
import InputField from '../../fields/InputField';
import validatePresence from '../../validators/validate_presence';

const RUN_TYPES = {
  RUN: 'RUN',
  RUN_AND_SAVE: 'RUN_AND_SAVE',
  SAVE: 'SAVE',
};

class SaveQueryForm extends Component {
  static propTypes = {
    onSubmit: PropTypes.func,
    saveQuery: PropTypes.bool,
  };

  constructor (props) {
    super(props);

    this.state = {
      errors: {
        name: null,
        description: null,
        duration: null,
        platforms: null,
        hosts: null,
        hostsPercentage: null,
        scanInterval: null,
      },
      formData: {
        name: null,
        description: null,
        duration: 'short',
        platforms: 'all',
        hosts: 'all',
        hostsPercentage: null,
        scanInterval: 0,
      },
      showMoreOptions: false,
    };
  }

  componentWillMount () {
    global.window.addEventListener('keydown', this.handleKeydown);
  }

  componentWillUnmount () {
    global.window.removeEventListener('keydown', this.handleKeydown);
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

  onFormSubmit = (runType) => {
    return (evt) => {
      if (evt) {
        evt.preventDefault();
      }

      const { formData } = this.state;
      const { onSubmit } = this.props;
      const { validate } = this;

      if (validate(runType)) return onSubmit({ formData, runType });

      return false;
    };
  }

  handleKeydown = (evt) => {
    const { metaKey, code } = evt;
    const { onFormSubmit } = this;
    const { RUN_AND_SAVE, RUN } = RUN_TYPES;
    const { saveQuery } = this.props;
    const runType = saveQuery ? RUN_AND_SAVE : RUN;

    if (metaKey && code === 'Enter') {
      return onFormSubmit(runType)();
    }

    return false;
  };

  validate = (runType) => {
    const {
      errors,
      formData: { name },
    } = this.state;
    const { RUN } = RUN_TYPES;

    if (runType === RUN) return true;

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

  toggleShowMoreOptions = () => {
    const { showMoreOptions } = this.state;

    this.setState({
      showMoreOptions: !showMoreOptions,
    });

    return false;
  };

  renderMoreOptionsCtaSection = () => {
    const { moreOptionsIconStyles, moreOptionsCtaSectionStyles, moreOptionsTextStyles } = componentStyles;
    const { showMoreOptions } = this.state;
    const { toggleShowMoreOptions } = this;

    if (showMoreOptions) {
      return (
        <div style={moreOptionsCtaSectionStyles}>
          <span onClick={toggleShowMoreOptions} style={moreOptionsTextStyles}>
            Fewer Options
            <i className="kolidecon-upcarat" style={moreOptionsIconStyles} />
          </span>
        </div>
      );
    }

    return (
      <div style={moreOptionsCtaSectionStyles}>
        <span onClick={toggleShowMoreOptions} style={moreOptionsTextStyles}>
          More Options
          <i className="kolidecon-downcarat" style={moreOptionsIconStyles} />
        </span>
      </div>
    );
  }

  renderMoreOptionsFormFields = () => {
    const {
      errors,
      formData: {
        duration,
        platforms,
        hosts,
      },
      showMoreOptions,
    } = this.state;
    const {
      dropdownInputStyles,
      formSectionStyles,
      helpTextStyles,
      labelStyles,
      queryDescriptionInputStyles,
      queryHostsPercentageStyles,
      queryNameInputStyles,
    } = componentStyles;
    const { onFieldChange } = this;

    if (!showMoreOptions) return false;

    return (
      <div>
        <div style={formSectionStyles}>
          <InputField
            error={errors.description}
            label="Query Description"
            labelStyles={labelStyles}
            name="description"
            onChange={onFieldChange('description')}
            placeholder="e.g. This query does x, y, & z because n"
            style={queryDescriptionInputStyles}
            type="textarea"
          />
          <small style={helpTextStyles}>
            If your query is really complex and/or it is not clear why you wrote this query, you should write a description so others can reuse this query for the correct reason.
          </small>
        </div>
        <div style={formSectionStyles}>
          <div>
            <label htmlFor="duration" style={labelStyles}>Query Duration</label>
            <select
              key="duration"
              name="duration"
              value={duration}
              onChange={onFieldChange('duration')}
              style={dropdownInputStyles}
            >
              <option value="short">Short</option>
              <option value="long">Long</option>
            </select>
          </div>
          <small style={helpTextStyles}>
            Individual hosts are not always online. A longer duration will return more complete results. You can view results of any in-progress query at any time.
          </small>
        </div>
        <div style={formSectionStyles}>
          <div>
            <label htmlFor="platforms" style={labelStyles}>Query Platform</label>
            <select
              key="platforms"
              name="platforms"
              value={platforms}
              onChange={onFieldChange('platforms')}
              style={dropdownInputStyles}
            >
              <option value="all">ALL PLATFORMS</option>
              <option value="none">NO PLATFORMS</option>
            </select>
          </div>
          <small style={helpTextStyles}>
            Specifying a platform allows you to restrict the query from running on a certain platform (even on hosts specifically targeted that do not match).
          </small>
        </div>
        <div style={formSectionStyles}>
          <div>
            <label htmlFor="hosts" style={labelStyles}>Run On All Hosts?</label>
            <div>
              <input
                checked={hosts === 'all'}
                onChange={onFieldChange('hosts')}
                type="radio"
                value="all"
              /> Run Query On All Hosts
              <br />
              <input
                checked={hosts === 'percentage'}
                onChange={onFieldChange('hosts')}
                type="radio"
                value="percentage"
              /> Run Query On
              <InputField
                inputWrapperStyles={{ display: 'inline-block' }}
                inputOptions={{ maxLength: 3 }}
                onChange={onFieldChange('hostsPercentage')}
                style={queryHostsPercentageStyles}
                type="tel"
              />% Of All Hosts
            </div>
          </div>
          <small style={helpTextStyles}>
            Specifying a platform allows you to restrict the query from running on a certain platform (even on hosts specifically targeted that do not match).
          </small>
        </div>
        <div style={formSectionStyles}>
          <InputField
            error={errors.scanInterval}
            label="Scan Interval (seconds)"
            labelStyles={labelStyles}
            name="scanInterval"
            onChange={onFieldChange('scanInterval')}
            placeholder="e.g. 300"
            style={queryNameInputStyles}
            type="tel"
          />
          <small style={helpTextStyles}>
            You can use queries you write in "scans". The interval can be used to control how frequently the query runs when it is running continuously.
          </small>
        </div>
      </div>
    );
  };

  renderRunQuery = () => {
    const {
      buttonStyles,
      runQuerySectionStyles,
      runQueryTipStyles,
    } = componentStyles;
    const { onFormSubmit } = this;
    const { RUN } = RUN_TYPES;

    return (
      <form style={runQuerySectionStyles} onSubmit={onFormSubmit(RUN)}>
        <span style={runQueryTipStyles}>&#8984; + Enter</span>
        <Button
          style={buttonStyles}
          text="Run Query"
          type="submit"
        />
      </form>
    );
  }

  render () {
    const {
      buttonInvertStyles,
      buttonStyles,
      buttonWrapperStyles,
      labelStyles,
      helpTextStyles,
      queryNameInputStyles,
      queryNameWrapperStyles,
    } = componentStyles;
    const { errors } = this.state;
    const {
      onFieldChange,
      onFormSubmit,
      renderMoreOptionsFormFields,
      renderMoreOptionsCtaSection,
      renderRunQuery,
    } = this;
    const { RUN_AND_SAVE, SAVE } = RUN_TYPES;
    const { saveQuery } = this.props;

    if (!saveQuery) return renderRunQuery();

    return (
      <form onSubmit={onFormSubmit(RUN_AND_SAVE)}>
        <div style={queryNameWrapperStyles}>
          <InputField
            error={errors.name}
            label="Query Name"
            labelStyles={labelStyles}
            name="name"
            onChange={onFieldChange('name')}
            placeholder="e.g. Interesting Query Name"
            style={queryNameInputStyles}
          />
          <small style={helpTextStyles}>
            Write a name that describes the query and its intent. Pick a name that others will find useful.
          </small>
        </div>
        {renderMoreOptionsCtaSection()}
        {renderMoreOptionsFormFields()}
        <div style={buttonWrapperStyles}>
          <Button
            onClick={onFormSubmit(SAVE)}
            style={buttonInvertStyles}
            text="Save Query Only"
            variant="inverse"
          />
          <Button
            text="Run & Save Query"
            type="submit"
            style={buttonStyles}
          />
        </div>
      </form>
    );
  }
}

export default radium(SaveQueryForm);
