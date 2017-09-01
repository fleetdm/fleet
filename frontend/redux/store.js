import Es6ObjectAssign from 'es6-object-assign';
import Es6Promise from 'es6-promise';

import { applyMiddleware, compose, createStore } from 'redux';
import { browserHistory } from 'react-router';
import { loadingBarMiddleware } from 'react-redux-loading-bar';
import { routerMiddleware } from 'react-router-redux';
import thunkMiddleware from 'redux-thunk';

import authMiddleware from './middlewares/auth';
import redirectMiddleware from './middlewares/redirect';
import reducers from './reducers';

// ie polyfills
Es6ObjectAssign.polyfill();
Es6Promise.polyfill();

const initialState = {};

const appliedMiddleware = applyMiddleware(
  thunkMiddleware,
  routerMiddleware(browserHistory),
  authMiddleware,
  redirectMiddleware,
  loadingBarMiddleware({
    promiseTypeSuffixes: ['REQUEST', 'SUCCESS', 'FAILURE'],
  }),
);

const composeEnhancers = process.env.NODE_ENV !== 'production' &&
  typeof global.window === 'object' &&
  global.window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__ ?
  global.window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__ : compose;
const store = createStore(
  reducers,
  initialState,
  composeEnhancers(appliedMiddleware),
);

export default store;
