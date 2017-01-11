import { PropTypes } from 'react';
import hostInterface from 'interfaces/host';
import labelInterface from 'interfaces/label';

export default PropTypes.oneOfType([hostInterface, labelInterface]);
