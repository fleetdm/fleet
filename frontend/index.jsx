import React from 'react';
import { Provider } from 'react-redux';
import { render } from 'react-dom';
import routes from './router';
import store from './redux/store';

if (typeof window !== 'undefined') {
  const { document } = global;
  const app = document.getElementById('app');
  const reactReduxApp = (
    <Provider store={store}>
      {routes}
    </Provider>
  );

  render(reactReduxApp, app);
}
