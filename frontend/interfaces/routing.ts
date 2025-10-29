// make up for some shortcomings of react router 3
export interface IRouterLocation {
  pathname: string;
  host: string; // hostname:port
  hostname: string;
  port: string;
  protocol: string;
  // lots of other stuff
  [key: string]: any;
}
