import moment from "moment";

export const convertSeconds = (totalMilliSeconds) => {
  return moment.utc(totalMilliSeconds).format("HH:mm:ss");
};

export default { convertSeconds };
