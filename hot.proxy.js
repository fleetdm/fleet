// Kolide Hot Proxy. Adapated From: https://github.com/olebedev/go-starter-kit/blob/c96a70a1ebf78ed9401dcfbee3f3c79093c28f8a/hot.proxy.js
//
// Hotproxy uses the following ENV variables to configure itself
//   KOLIDE_DEV_PROXY_PORT - Port where kolide binary server is listening (Default: 8080)
//   KOLIDE_DEV_PROXY_HOST - Hostname or IP where kolide binary server is bound (Default: localhost)
//   KOLIDE DEV_PROXY_LISTEN_PORT - Port where Kolide hot proxy listens (Default: KOLIDE_DEV_PROXY_PORT + 1)
//

let webpack = require('webpack');
let webpackDevMiddleware = require('webpack-dev-middleware');
let webpackHotMiddleware = require('webpack-hot-middleware');
let proxy = require('proxy-middleware');
let config = require('./webpack.config');
let url = require('url');

let proxyPort = +(process.env.KOLIDE_DEV_PROXY_PORT || 8080);
let proxyHost = (process.env.KOLIDE_DEV_PROXY_HOST || 'localhost');
let proxyListenPort = +(process.env.DEV_PROXY_LISTEN_PORT || proxyPort + 1 );

config.entry = {
  bundle: [
    'webpack-hot-middleware/client?https://' + proxyHost + ':' + proxyPort,
    config.entry.bundle
  ]
};

config.plugins.push(
  new webpack.optimize.OccurenceOrderPlugin(),
  new webpack.HotModuleReplacementPlugin(),
  new webpack.NoErrorsPlugin()
);

config.devtool = 'cheap-module-eval-source-map';

let proxyOptions = url.parse('https://' + proxyHost + ':' + proxyPort);
proxyOptions.rejectUnauthorized = false // Disables self-signed cert checking

let app = new require('express')();

let compiler = webpack(config);
app.use(webpackDevMiddleware(compiler, { noInfo: true, publicPath: config.output.publicPath }));
app.use(webpackHotMiddleware(compiler));
app.use(proxy(proxyOptions));

app.listen(proxyListenPort, function(error) {
  if (error) {
    console.error(error);
  } else {
    console.info("==> 🌎  Listening on port %s. Open up http://localhost:%s/ in your browser.", proxyListenPort, proxyListenPort);
  }
});
