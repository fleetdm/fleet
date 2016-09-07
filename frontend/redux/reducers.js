import { combineReducers } from 'redux';
import { routerReducer } from 'react-router-redux';
import app from './nodes/app/reducer';

export default combineReducers({
  app,
  routing: routerReducer,
});
