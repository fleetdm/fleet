import ReactDOM from 'react-dom';
import routes from './router';

if (typeof window !== 'undefined') {
  const { document } = global;
  const app = document.getElementById('app');

  ReactDOM.render(routes, app);
}
