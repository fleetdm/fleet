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

const store = createStore(
  reducers,
  initialState,
  compose(appliedMiddleware),
);

export default store;
