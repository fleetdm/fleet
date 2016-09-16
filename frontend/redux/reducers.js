import { combineReducers } from 'redux';
import { routerReducer } from 'react-router-redux';
import app from './nodes/app/reducer';
import auth from './nodes/auth/reducer';
import components from './nodes/components/reducer';

export default combineReducers({
  app,
  auth,
  components,
  routing: routerReducer,
});
