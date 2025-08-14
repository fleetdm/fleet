module.exports = {

  friendlyName: 'Calculate time excluding weekends',

  description: 'Calculates time difference between two dates excluding weekends if enabled.',

  extendedDescription: 'This function is based on https://github.com/fleetdm/fleet/blob/d20ddf33280464b1377aba8f755eb74df2f72724/.github/actions/eng-metrics/src/github-client.js#L512, where it is thoroughly unit tested.',

  inputs: {
    startTime: {
      type: 'ref',
      description: 'The start time',
      required: true
    },
    endTime: {
      type: 'ref',
      description: 'The end time',
      required: true
    }
  },

  exits: {
    success: {
      description: 'Successfully calculated time difference.',
      outputType: 'number'
    }
  },

  fn: async function ({ startTime, endTime }) {
    if (!sails.config.custom.githubProjectsV2.excludeWeekends) {
      // If weekend exclusion is disabled, return simple time difference
      return Math.floor((endTime - startTime) / 1000);
    }

    // Use the provided weekend exclusion logic
    const startDay = startTime.getUTCDay();
    const endDay = endTime.getUTCDay();

    // Case: Both start time and end time are on the same weekend
    if (
      (startDay === 0 || startDay === 6) &&
      (endDay === 0 || endDay === 6) &&
      Math.floor(endTime / (24 * 60 * 60 * 1000)) -
      Math.floor(startTime / (24 * 60 * 60 * 1000)) <=
      2
    ) {
      // Return 0 seconds
      return 0;
    }

    // Make copies to avoid modifying original dates
    const adjustedStartTime = new Date(startTime);
    const adjustedEndTime = new Date(endTime);

    // Set to start of Monday if start time is on weekend
    if (startDay === 0) {
      // Sunday
      adjustedStartTime.setUTCDate(adjustedStartTime.getUTCDate() + 1);
      adjustedStartTime.setUTCHours(0, 0, 0, 0);
    } else if (startDay === 6) {
      // Saturday
      adjustedStartTime.setUTCDate(adjustedStartTime.getUTCDate() + 2);
      adjustedStartTime.setUTCHours(0, 0, 0, 0);
    }

    // Set to start of Saturday if end time is on Sunday
    if (endDay === 0) {
      // Sunday
      adjustedEndTime.setUTCDate(adjustedEndTime.getUTCDate() - 1);
      adjustedEndTime.setUTCHours(0, 0, 0, 0);
    } else if (endDay === 6) {
      // Saturday
      adjustedEndTime.setUTCHours(0, 0, 0, 0);
    }

    // Calculate raw time difference in milliseconds
    const weekendDays = countWeekendDays(adjustedStartTime, adjustedEndTime);
    const diffMs = adjustedEndTime - adjustedStartTime - weekendDays * 24 * 60 * 60 * 1000;

    // Ensure we don't return negative values
    return Math.max(0, Math.floor(diffMs / 1000));

    /**
     * Counts the number of weekend days between two dates (inlined helper function)
     *
     * @param {Date} startDate - The start date
     * @param {Date} endDate - The end date
     * @returns {number} Number of weekend days
     */
    function countWeekendDays(startDate, endDate) {
      // Make local copies of dates
      startDate = new Date(startDate);
      endDate = new Date(endDate);

      // Ensure startDate is before endDate
      if (startDate > endDate) {
        [startDate, endDate] = [endDate, startDate];
      }

      // Make sure start dates and end dates are not on weekends. We just want to count the weekend days between them.
      if (startDate.getUTCDay() === 0) {
        startDate.setUTCDate(startDate.getUTCDate() + 1);
      } else if (startDate.getUTCDay() === 6) {
        startDate.setUTCDate(startDate.getUTCDate() + 2);
      }
      if (endDate.getUTCDay() === 0) {
        endDate.setUTCDate(endDate.getUTCDate() - 2);
      } else if (endDate.getUTCDay() === 6) {
        endDate.setUTCDate(endDate.getUTCDate() - 1);
      }

      let count = 0;
      const current = new Date(startDate);

      while (current <= endDate) {
        const day = current.getUTCDay();
        if (day === 0 || day === 6) {
          // Sunday (0) or Saturday (6)
          count++;
        }
        current.setUTCDate(current.getUTCDate() + 1);
      }

      return count;
    }
  }

};
