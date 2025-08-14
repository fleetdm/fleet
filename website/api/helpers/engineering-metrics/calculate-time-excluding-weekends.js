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

    // Count weekend days between adjusted dates
    // Make local copies for weekend counting
    let weekendStartDate = new Date(adjustedStartTime);
    let weekendEndDate = new Date(adjustedEndTime);

    // Ensure weekendStartDate is before weekendEndDate
    if (weekendStartDate > weekendEndDate) {
      [weekendStartDate, weekendEndDate] = [weekendEndDate, weekendStartDate];
    }

    // Make sure start dates and end dates are not on weekends. We just want to count the weekend days between them.
    if (weekendStartDate.getUTCDay() === 0) {
      weekendStartDate.setUTCDate(weekendStartDate.getUTCDate() + 1);
    } else if (weekendStartDate.getUTCDay() === 6) {
      weekendStartDate.setUTCDate(weekendStartDate.getUTCDate() + 2);
    }
    if (weekendEndDate.getUTCDay() === 0) {
      weekendEndDate.setUTCDate(weekendEndDate.getUTCDate() - 2);
    } else if (weekendEndDate.getUTCDay() === 6) {
      weekendEndDate.setUTCDate(weekendEndDate.getUTCDate() - 1);
    }

    let weekendDays = 0;
    const current = new Date(weekendStartDate);

    while (current <= weekendEndDate) {
      const day = current.getUTCDay();
      if (day === 0 || day === 6) {
        // Sunday (0) or Saturday (6)
        weekendDays++;
      }
      current.setUTCDate(current.getUTCDate() + 1);
    }

    // Calculate raw time difference in milliseconds
    const diffMs = adjustedEndTime - adjustedStartTime - weekendDays * 24 * 60 * 60 * 1000;

    // Ensure we don't return negative values
    return Math.max(0, Math.floor(diffMs / 1000));
  }

};
