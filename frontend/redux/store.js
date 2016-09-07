import { createStore, applyMiddleware, compose } from 'redux';
import thunkMiddleware from 'redux-thunk';
import { browserHistory } from 'react-router';
import { routerMiddleware } from 'react-router-redux';
import reducers from './reducers';

const initialState = {};
const middleware = [
  thunkMiddleware,
  routerMiddleware(browserHistory),
];
const appliedMiddleware = applyMiddleware(...middleware);

const store = createStore(
  reducers,
  initialState,
  compose(appliedMiddleware),
);
export default store;
