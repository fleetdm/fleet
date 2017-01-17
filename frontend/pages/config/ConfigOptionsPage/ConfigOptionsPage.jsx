import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { differenceWith, find, filter, isEqual, noop } from 'lodash';

import Button from 'components/buttons/Button';
import configOptionActions from 'redux/nodes/entities/config_options/actions';
import ConfigOptionsForm from 'components/forms/ConfigOptionsForm';
import Icon from 'components/icons/Icon';
import configOptionInterface from 'interfaces/config_option';
import debounce from 'utilities/debounce';
import defaultConfigOptions from 'pages/config/ConfigOptionsPage/default_config_options';
import entityGetter from 'redux/utilities/entityGetter';
import helpers from 'pages/config/ConfigOptionsPage/helpers';
import { renderFlash } from 'redux/nodes/notifications/actions';

const baseClass = 'config-options-page';
const DEFAULT_CONFIG_OPTION = { name: '', value: '' };

export class ConfigOptionsPage extends Component {
  static propTypes = {
    configOptions: PropTypes.arrayOf(configOptionInterface),
    dispatch: PropTypes.func.isRequired,
  };

  static defaultProps = {
    configOptions: [],
    dispatch: noop,
  };

  constructor (props) {
    super(props);

    this.state = {
      configOptions: [],
      configOptionErrors: {},
    };
  }

  componentWillMount () {
    const { configOptions, dispatch } = this.props;

    this.setState({ configOptions });

    dispatch(configOptionActions.loadAll());

    return false;
  }

  componentWillReceiveProps ({ configOptions }) {
    if (!isEqual(configOptions, this.state.configOptions)) {
      this.setState({ configOptions });
    }

    return false;
  }

  onAddNewOption = (evt) => {
    evt.preventDefault();

    const { configOptions } = this.state;

    if (find(configOptions, DEFAULT_CONFIG_OPTION)) {
      return false;
    }

    this.setState({
      configOptions: [
        ...configOptions,
        DEFAULT_CONFIG_OPTION,
      ],
    });

    return false;
  }

  onOptionUpdate = (oldOption, newOption) => {
    const { configOptions } = this.state;
    const newConfigOptions = helpers.updatedConfigOptions({ oldOption, newOption, configOptions });

    this.setState({ configOptions: newConfigOptions });

    return false;
  }

  onRemoveOption = (option) => {
    const { configOptions } = this.state;
    const configOptionsWithoutRemovedOption = filter(configOptions, o => !isEqual(o, option));

    if (isEqual(option, DEFAULT_CONFIG_OPTION)) {
      this.setState({ configOptions: configOptionsWithoutRemovedOption });
    } else {
      this.setState({
        configOptions: [
          ...configOptionsWithoutRemovedOption,
          { ...option, value: null },
        ],
      });
    }

    return false;
  }

  onResetConfigOptions = () => {
    this.setState({ configOptions: defaultConfigOptions });

    return false;
  }

  onSave = debounce(() => {
    const { dispatch } = this.props;
    const changedOptions = this.calculateChangedOptions();
    const { errors, valid } = this.validate();

    if (!changedOptions.length) {
      return false;
    }

    if (!valid) {
      this.setState({ configOptionErrors: errors });

      return false;
    }

    const formattedChangedOptions = helpers.formatOptionsForServer(changedOptions);

    dispatch(configOptionActions.update(formattedChangedOptions))
      .then(() => {
        dispatch(renderFlash('success', 'Options updated!'));

        return false;
      })
      .catch(() => {
        dispatch(renderFlash('error', 'We were unable to update your config options'));
        return false;
      });

    return false;
  })

  calculateChangedOptions = () => {
    const { configOptions: stateConfigOptions } = this.state;
    const { configOptions: propConfigOptions } = this.props;
    const presentStateConfigOptions = filter(stateConfigOptions, o => o.name);

    return differenceWith(presentStateConfigOptions, propConfigOptions, isEqual);
  }

  validate = () => {
    const { configOptions: allConfigOptions } = this.state;
    const changedConfigOptions = this.calculateChangedOptions();

    return helpers.configErrorsFor(changedConfigOptions, allConfigOptions);
  }

  render () {
    const { configOptionErrors, configOptions } = this.state;
    const { onAddNewOption, onOptionUpdate, onRemoveOption, onResetConfigOptions, onSave } = this;
    const availableOptions = filter(configOptions, option => option.value !== null);

    return (
      <div className={`body-wrap ${baseClass}`}>
        <div className={`${baseClass}__header-wrapper`}>
          <div className={`${baseClass}__header-content`}>
            <h1>Manage Additional Osquery Options</h1>
            <p>
              Osquery allows you to set a number of configuration options (<a href="https://osquery.io/docs/" target="_blank" rel="noopener noreferrer">Osquery Documentation</a>).
              Since Kolide manages your Osquery configuration, you can set these additional desired
              options on this screen. Some options that Kolide needs to function correctly will be ignored.
            </p>
          </div>
          <div className={`${baseClass}__btn-wrapper`}>
            <Button block className={`${baseClass}__reset-btn`} onClick={onResetConfigOptions} variant="inverse">
              RESET TO DEFAULT
            </Button>
            <Button block className={`${baseClass}__save-btn`} onClick={onSave} variant="brand">
              SAVE OPTIONS
            </Button>
          </div>
        </div>
        <ConfigOptionsForm
          configNameOptions={helpers.configOptionDropdownOptions(configOptions)}
          completedOptions={availableOptions}
          errors={configOptionErrors}
          onFormUpdate={onOptionUpdate}
          onRemoveOption={onRemoveOption}
        />
        <Button onClick={onAddNewOption} variant="unstyled" className={`${baseClass}__add-new`}><Icon name="add-plus" /> Add New Option</Button>
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { entities: configOptions } = entityGetter(state).get('config_options');

  return { configOptions };
};

export default connect(mapStateToProps)(ConfigOptionsPage);
