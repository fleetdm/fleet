import { createStore, applyMiddleware, compose } from 'redux';
import thunkMiddleware from 'redux-thunk';
import { browserHistory } from 'react-router';
import { routerMiddleware } from 'react-router-redux';

import authMiddleware from './middlewares/auth';
import reducers from './reducers';

const initialState = {};

const appliedMiddleware = applyMiddleware(
  thunkMiddleware,
  routerMiddleware(browserHistory),
  authMiddleware,
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
