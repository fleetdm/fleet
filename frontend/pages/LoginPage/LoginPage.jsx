import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { push } from 'react-router-redux';
import ReactCSSTransitionGroup from 'react-addons-css-transition-group';
import debounce from '../../utilities/debounce';
import local from '../../utilities/local';
import LoginForm from '../../components/forms/LoginForm';
import LoginSuccessfulPage from '../LoginSuccessfulPage';
import { loginUser } from '../../redux/nodes/auth/actions';
import paths from '../../router/paths';
import './styles.scss';


export class LoginPage extends Component {

  static propTypes = {
    dispatch: PropTypes.func,
    error: PropTypes.string,
    loading: PropTypes.bool,
    user: PropTypes.object,
  };

  constructor () {
    super();
    this.state = {
      loginVisible: true,
    };
  }

  componentWillMount () {
    const { dispatch } = this.props;

    if (local.getItem('auth_token')) {
      return dispatch(push('/'));
    }

    return false;
  }

  onSubmit = debounce((formData) => {
    const { dispatch } = this.props;
    const { HOME } = paths;
    const redirectTime = 1500;
    return dispatch(loginUser(formData))
      .then(() => {
        this.setState({ loginVisible: false });
        setTimeout(() => {
          return dispatch(push(HOME));
        }, redirectTime);
      });
  })

  render () {
    const { onSubmit } = this;
    const { loginVisible } = this.state;

    return (
      <div>
        <LoginSuccessfulPage />
        <ReactCSSTransitionGroup
          transitionName="login-form-animation"
          transitionEnterTimeout={500}
          transitionLeaveTimeout={300}
        >
          {loginVisible && <LoginForm onSubmit={onSubmit} visible={loginVisible} />}
        </ReactCSSTransitionGroup>
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { error, loading, user } = state.auth;

  return {
    error,
    loading,
    user,
  };
};

export default connect(mapStateToProps)(LoginPage);
