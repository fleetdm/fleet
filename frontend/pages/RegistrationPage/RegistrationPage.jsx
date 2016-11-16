import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { noop } from 'lodash';

import Breadcrumbs from 'pages/RegistrationPage/Breadcrumbs';
import RegistrationForm from 'components/forms/RegistrationForm';
import { showBackgroundImage } from 'redux/nodes/app/actions';

export class RegistrationPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func.isRequired,
  };

  static defaultProps = {
    dispatch: noop,
  };

  constructor (props) {
    super(props);

    this.state = { page: 1 };

    return false;
  }

  componentWillMount () {
    const { dispatch } = this.props;

    dispatch(showBackgroundImage);

    return false;
  }

  onNextPage = () => {
    const { page } = this.state;
    this.setState({ page: page + 1 });

    return false;
  }

  onRegistrationFormSubmit = (formData) => {
    console.log('registration form submitted:', formData);

    return false;
  }

  onSetPage = (page) => {
    this.setState({ page });

    return false;
  }

  render () {
    const { page } = this.state;
    const { onRegistrationFormSubmit, onNextPage, onSetPage } = this;

    return (
      <div>
        <Breadcrumbs onClick={onSetPage} page={page} />
        <RegistrationForm page={page} onNextPage={onNextPage} onSubmit={onRegistrationFormSubmit} />
      </div>
    );
  }
}

export default connect()(RegistrationPage);
