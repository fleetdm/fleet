import React from 'react';
import PropTypes from 'prop-types';
import { useDispatch } from 'react-redux';
import { push } from 'react-router-redux';

import hostInterface from 'interfaces/host';
import helpers from 'kolide/helpers';
import PATHS from 'router/paths';
import Button from 'components/buttons/Button';

const LinkCell = (props) => {
  const { value, host } = props;

  const dispatch = useDispatch();

  const onHostClick = (selectedHost) => {
    dispatch(push(PATHS.HOST_DETAILS(selectedHost)));
  };

  const lastSeenTime = (status, seenTime) => {
    const { humanHostLastSeen } = helpers;

    if (status !== 'online') {
      return `Last Seen: ${humanHostLastSeen(seenTime)} UTC`;
    }

    return 'Online';
  };

  return (
    <Button
      onClick={() => onHostClick(host)}
      variant="text-link"
      title={lastSeenTime(host.status, host.seen_time)}
    >
      {value}
    </Button>
  );
};

LinkCell.propTypes = {
  value: PropTypes.string,
  host: hostInterface,
};

export default LinkCell;
