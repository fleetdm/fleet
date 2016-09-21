import React from 'react';
import { browserHistory, IndexRoute, Route, Router } from 'react-router';
import { Provider } from 'react-redux';
import radium from 'radium';
import { syncHistoryWithStore } from 'react-router-redux';
import AdminDashboardPage from '../pages/Admin/DashboardPage';
import App from '../components/App';
import AuthenticatedAdminRoutes from '../components/AuthenticatedAdminRoutes';
import AuthenticatedRoutes from '../components/AuthenticatedRoutes';
import CoreLayout from '../layouts/CoreLayout';
import ForgotPasswordPage from '../pages/ForgotPasswordPage';
import HomePage from '../pages/HomePage';
import LoginPage from '../pages/LoginPage';
import LoginSuccessfulPage from '../pages/LoginSuccessfulPage';
import LoginRoutes from '../components/LoginRoutes';
import LogoutPage from '../pages/LogoutPage';
import ResetPasswordPage from '../pages/ResetPasswordPage';
import store from '../redux/store';

const history = syncHistoryWithStore(browserHistory, store);

const routes = (
  <Provider store={store}>
    <Router history={history}>
      <Route path="/" component={radium(App)}>
        <Route component={radium(LoginRoutes)}>
          <Route path="forgot_password" component={radium(ForgotPasswordPage)} />
          <Route path="login" component={radium(LoginPage)} />
          <Route path="reset_password" component={radium(ResetPasswordPage)} />
          <Route component={AuthenticatedRoutes}>
            <Route path="login_successful" component={radium(LoginSuccessfulPage)} />
            <Route path="logout" component={radium(LogoutPage)} />
          </Route>
        </Route>
        <Route component={AuthenticatedRoutes}>
          <Route component={radium(CoreLayout)}>
            <IndexRoute component={radium(HomePage)} />
            <Route path="admin" component={AuthenticatedAdminRoutes}>
              <IndexRoute component={radium(AdminDashboardPage)} />
            </Route>
          </Route>
        </Route>
      </Route>
    </Router>
  </Provider>
);

export default routes;
